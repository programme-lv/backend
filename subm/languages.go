package subm

import (
	"context"

	submgen "github.com/programme-lv/backend/gen/submissions"
)

// ListProgrammingLanguages implements submissions.Service.
func (s *SubmissionsService) ListProgrammingLanguages(context.Context) (res []*submgen.ProgrammingLang, err error) {
	res = make([]*submgen.ProgrammingLang, 0)
	langs := getHardcodedLanguageList()
	for _, lang := range langs {
		res = append(res, &submgen.ProgrammingLang{
			ID:               lang.ID,
			FullName:         lang.FullName,
			CodeFilename:     &lang.CodeFilename,
			CompileCmd:       lang.CompileCmd,
			ExecuteCmd:       lang.ExecuteCmd,
			EnvVersionCmd:    lang.EnvVersionCmd,
			HelloWorldCode:   lang.HelloWorldCode,
			MonacoID:         lang.MonacoId,
			CompiledFilename: lang.CompiledFilename,
			Enabled:          lang.Enabled,
		})
	}
	return res, nil
}

// ProgrammingLang represents the structure of a programming language.
type ProgrammingLang struct {
	ID               string
	FullName         string
	CodeFilename     string
	CompileCmd       *string
	ExecuteCmd       string
	EnvVersionCmd    string
	HelloWorldCode   string
	MonacoId         string
	CompiledFilename *string
	Enabled          bool
}

// getHardcodedLanguageList returns a list of hardcoded programming languages.
func getHardcodedLanguageList() []ProgrammingLang {
	languages := []ProgrammingLang{
		{
			ID:               "python3.10",
			FullName:         "Python 3.10",
			CodeFilename:     "main.py",
			CompileCmd:       nil,
			ExecuteCmd:       "python3.10 main.py",
			EnvVersionCmd:    "python3.10 --version",
			HelloWorldCode:   `print("Hello, World!")`,
			MonacoId:         "python",
			CompiledFilename: nil,
			Enabled:          false,
		},
		{
			ID:               "python3.11",
			FullName:         "Python 3.11",
			CodeFilename:     "main.py",
			CompileCmd:       nil,
			ExecuteCmd:       "python3.11 main.py",
			EnvVersionCmd:    "python3.11 --version",
			HelloWorldCode:   `print("Hello, World!")`,
			MonacoId:         "python",
			CompiledFilename: nil,
			Enabled:          true,
		},
		{
			ID:            "go1.19",
			FullName:      "Go 1.19",
			CodeFilename:  "main.go",
			CompileCmd:    strPtr("go build main.go"),
			ExecuteCmd:    "./main",
			EnvVersionCmd: "go version",
			HelloWorldCode: `package main
import "fmt"
func main() {
    fmt.Println("Hello, World!")
}`,
			MonacoId:         "go",
			CompiledFilename: strPtr("main"),
			Enabled:          false,
		},
		{
			ID:            "go1.21",
			FullName:      "Go 1.21",
			CodeFilename:  "main.go",
			CompileCmd:    strPtr("go build main.go"),
			ExecuteCmd:    "./main",
			EnvVersionCmd: "go version",
			HelloWorldCode: `package main
import "fmt"
func main() {
    fmt.Println("Hello, World!")
}`,
			MonacoId:         "go",
			CompiledFilename: strPtr("main"),
			Enabled:          true,
		},
		{
			ID:            "java21",
			FullName:      "Java SE 21",
			CodeFilename:  "Main.java",
			CompileCmd:    strPtr("javac Main.java"),
			ExecuteCmd:    "java -Xss64M -Xmx1024M -Xms8M -XX:NewRatio=2 -XX:TieredStopAtLevel=1 -XX:+UseSerialGC Main",
			EnvVersionCmd: "java --version",
			HelloWorldCode: `public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
    }
}`,
			MonacoId:         "java",
			CompiledFilename: strPtr("Main.class"),
			Enabled:          true,
		},
		{
			ID:            "cpp17",
			FullName:      "C++17 (GCC)",
			CodeFilename:  "main.cpp",
			CompileCmd:    strPtr("g++ -std=c++17 -o main main.cpp"),
			ExecuteCmd:    "./main",
			EnvVersionCmd: "g++ --version",
			HelloWorldCode: `#include <iostream>
int main() { std::cout << "Hello, World!" << std::endl; }`,
			MonacoId:         "cpp",
			CompiledFilename: strPtr("main"),
			Enabled:          true,
		},
	}

	return languages
}

// strPtr is a helper function to create a pointer to a string literal.
func strPtr(s string) *string {
	return &s
}
