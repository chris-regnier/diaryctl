package shell

import (
	"fmt"
	"io"
)

// WriteBashInit writes the bash shell integration script to the writer.
func WriteBashInit(w io.Writer) {
	fmt.Fprint(w, `# diaryctl shell integration
__diaryctl_prompt_hook() {
  eval "$(command diaryctl status --env 2>/dev/null)"
}

diaryctl_prompt_info() {
  command diaryctl status 2>/dev/null
}

if [[ -z "$PROMPT_COMMAND" ]]; then
  PROMPT_COMMAND="__diaryctl_prompt_hook"
else
  PROMPT_COMMAND="__diaryctl_prompt_hook;${PROMPT_COMMAND}"
fi

eval "$(command diaryctl completion bash 2>/dev/null)"
`)
}

// WriteZshInit writes the zsh shell integration script to the writer.
func WriteZshInit(w io.Writer) {
	fmt.Fprint(w, `# diaryctl shell integration
__diaryctl_prompt_hook() {
  eval "$(command diaryctl status --env 2>/dev/null)"
}

diaryctl_prompt_info() {
  command diaryctl status 2>/dev/null
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd __diaryctl_prompt_hook

eval "$(command diaryctl completion zsh 2>/dev/null)"
`)
}
