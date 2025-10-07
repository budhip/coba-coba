package errorgen

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"go/format"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/Masterminds/sprig"
	"github.com/iancoleman/strcase"
)

type (
	ErrorGen struct {
		ErrorMaps     []ErrorMap
		ErrorKeys     []ErrorKey
		ErrorMessages []ErrorMessage
		ErrorCodes    []ErrorCode
	}

	ErrorMap struct {
		Key     string
		Code    string
		Message string
	}

	ErrorKey struct {
		Key         string
		Description string
	}

	ErrorMessage struct {
		Key         string
		Description string
	}

	ErrorCode struct {
		Key         string
		Description string
	}
)

func GenerateErrorMapFromCSV(
	templateFileDir,
	templateName,
	fileLocation,
	outputDestination,
	outputFile string,
) {
	csvFile, err := os.Open(fileLocation)
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()

	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	var (
		isExistErrorMessage = make(map[string]bool)
		isExistErrorCode    = make(map[string]bool)
		data                ErrorGen
	)
	for i := 1; i < len(csvLines); i++ {
		key := csvLines[i][0]
		code := csvLines[i][1]
		message := csvLines[i][2]

		keyToSnake := strcase.ToCamel(key)
		errKey := fmt.Sprintf("%s%s", "ErrKey", strings.Join(strings.Split(keyToSnake, " "), ""))

		data.ErrorKeys = append(data.ErrorKeys, ErrorKey{
			Key:         errKey,
			Description: key,
		})

		codeCamel := strcase.ToCamel(code)
		errCodeKey := fmt.Sprintf("%s%s", "errCode", strings.Join(strings.Split(codeCamel, " "), ""))
		if ok := isExistErrorCode[errCodeKey]; !ok {
			data.ErrorCodes = append(data.ErrorCodes, ErrorCode{
				Key:         errCodeKey,
				Description: code,
			})
		}
		isExistErrorCode[errCodeKey] = true

		messageCamel := strcase.ToCamel(message)
		errMessageKey := fmt.Sprintf("%s%s", "err", strings.Join(strings.Split(messageCamel, " "), ""))

		if ok := isExistErrorMessage[errMessageKey]; !ok {
			data.ErrorMessages = append(data.ErrorMessages, ErrorMessage{
				Key:         errMessageKey,
				Description: message,
			})
		}
		isExistErrorMessage[errMessageKey] = true

		data.ErrorMaps = append(data.ErrorMaps, ErrorMap{
			Key:     errKey,
			Code:    errCodeKey,
			Message: errMessageKey,
		})

	}

	var processed bytes.Buffer
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	templateFileDir = wd + templateFileDir

	tmpl := template.Must(template.New("").Funcs(sprig.FuncMap()).ParseFiles(templateFileDir))
	if err := tmpl.ExecuteTemplate(&processed, templateName, data); err != nil {
		log.Fatalf("unable to parse data into template: %v\n", err)
	}

	formatted, err := format.Source(processed.Bytes())
	if err != nil {
		log.Fatalf("could not format processed template: %v\n", err)
	}

	outputPath := outputDestination + outputFile
	log.Printf("writing file: %s", outputPath)

	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("unable to create file: %v\n", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err = w.WriteString(string(formatted)); err != nil {
		log.Fatal(err)
	}
	w.Flush()
}
