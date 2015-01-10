rewrite = {};

rewrite.init = function() {
    var refresh = function(commit) {
	var suffixes = file_suffixes.val().trim().split(/\s+/);
	if (suffixes.length == 1 && suffixes[0] == "") {
	    suffixes = [];
	}

	$.ajax("/query", {
	    method: "POST",
	    data: JSON.stringify({
		Directory: directory.val(),
		FileSuffixes: suffixes,
		MatchContent: content_regex.val(),
		ReplaceContent: content_replace.val(),
		Commit: !!commit,
	    }),
	    dataType: "json",
	    success: function(data, status) {
		output.empty();

		if (data.Error) {
		    output.append('<div>' + data.Error + '</div>');
		}

		output.append('<div>' + data.Files.length + ' files</div>');

		for (i in data.Files) {
		    var f = data.Files[i];
		    output.append('<div><b>' + f.Path + '</b></div>');
		    if (f.Lines.length) {
			var container = $('<div class="filediff"/>');
			for (j in f.Lines) {
			    var line = f.Lines[j];
			    console.log(line);
			    var original = $('<div><span class="lineno">' + (line.Line + 1) + '</span><span class="code">' + line.Text + '</span></div>');
			    original.css('color', '#040');
			    container.append(original);
			    if (line.Rewritten) {
				var rewritten = $('<div><pre>' + line.Rewritten + '</pre></div>');
				rewritten.css('color', '#400');
				container.append(rewritten);
			    }
			}
			output.append(container);
		    }
		}
	    },
	});
    };

    var refreshOnEnter = function(e) {
	e.keyup(function(event) {
	    if (event.keyCode == 13) {
		refresh(false);
	    }
	});
    };

    var form = $('<form>');

    var directory = $('<input type="text" name"Directory" size=45 placeholder="Directory" />');
    refreshOnEnter(directory);
    form.append(directory);

    var file_suffixes = $('<input type="text" name="FileSuffixes" size=45 placeholder="File sufixes" />');
    refreshOnEnter(file_suffixes);
    form.append(file_suffixes);

    var content_regex = $('<input type="text" name="MatchContent" size=100 placeholder="Content regex" />');
    refreshOnEnter(content_regex);
    var div = $('<div />');
    div.append(content_regex);
    form.append(div);

    var content_replace = $('<input type="text" name="MatchContent" size=100 placeholder="Content replacement" />');
    refreshOnEnter(content_replace);
    div = $('<div />');
    div.append(content_replace);
    form.append(div);


    var button = $('<input type="button" value="Refresh" />');
    button.click(function() { refresh(false); });
    form.append(button);

    var button = $('<input type="button" value="Commit" />');
    button.click(function() { refresh(true); });
    form.append(button);

    var output = $('<div />');

    $('body').append(form).append(output);

};
