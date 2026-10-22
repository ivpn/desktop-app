package openvpn

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClientUpDoesNotEvalForeignOption(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "References", "Linux", "etc", "client.up")
	body, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "eval fopt=") {
		t.Fatal("client.up still evals foreign_option without quoting")
	}
	if !strings.Contains(text, `eval "fopt=\$(printf '%s' \"\${foreign_option_${i}}\")"`) {
		t.Fatal("client.up is missing the quoted foreign_option read")
	}

	out, err := exec.Command("sh", "-c", `
i=1
foreign_option_1='dhcp-option DNS $(echo PWNED)'
marker=/tmp/ivpn-foreign-option-test-$$
rm -f "$marker"
eval "fopt=\$(printf '%s' \"\${foreign_option_${i}}\")"
case "$fopt" in
  dhcp-option\ DNS\ *)
    printf '%s' "${fopt#dhcp-option DNS }"
    ;;
esac
if [ -f "$marker" ]; then printf 'RAN'; fi
`).Output()
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if got != "$(echo PWNED)" {
		t.Fatalf("parsed value = %q", got)
	}
}
