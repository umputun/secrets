# fish completion for secrets (generated via go-flags)
complete -c secrets -a '(GO_FLAGS_COMPLETION=verbose secrets (commandline -cop) 2>/dev/null | string replace -r "\\s+# " "\t")'
