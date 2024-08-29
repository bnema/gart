package system

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PromptForGitVersioning asks the user if they want to enable Git versioning
func PromptForGitVersioning() (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Would you like to enable git versioning? (y/n): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
