package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/comprehendmedical"
	"github.com/aws/aws-sdk-go/service/textract"
)

var linebreak = "\r\n"

func main() {
	defer func() {
		if oof := recover(); oof != nil {
			fmt.Printf("%+v", oof)
		}
	}()

	session := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	},
	))

	text, err := extractText(session)
	errorCheck(err)
	results, err := compMedical(session, text)
	errorCheck(err)

	file, err := os.Create("go_results.txt")
	defer file.Close()
	errorCheck(err)
	file.WriteString(text + linebreak + linebreak)
	file.WriteString(results)
}

func extractText(session *session.Session) (text string, err error) {
	// var s3obj textract.S3Object
	var document textract.Document
	var analyze textract.AnalyzeDocumentInput
	// s3obj.SetBucket("just-this-bucket-you-know")
	// s3obj.SetName("test_file.png")
	// document.SetS3Object(&s3obj)

	file, _ := os.Open("test_file.png")
	reader := bufio.NewReader(file)
	image, _ := ioutil.ReadAll(reader)

	document.SetBytes(image)
	analyze.SetDocument(&document)
	forms := textract.FeatureTypeForms
	tables := textract.FeatureTypeTables
	analyze.SetFeatureTypes([]*string{&forms, &tables})

	service := textract.New(session)
	output, err := service.AnalyzeDocument(&analyze)
	if err != nil {
		return
	}

	for i := range output.Blocks {
		block := output.Blocks[i]
		if *block.BlockType == textract.BlockTypeWord {
			text += " " + *block.Text
		}
	}

	return
}

func compMedical(session *session.Session, text string) (results string, err error) {
	service := comprehendmedical.New(session)
	var dei comprehendmedical.DetectEntitiesV2Input
	var icd10i comprehendmedical.InferICD10CMInput
	var rxni comprehendmedical.InferRxNormInput
	dei.SetText(text)
	icd10i.SetText(text)
	rxni.SetText(text)

	entities, err := service.DetectEntitiesV2(&dei)
	if err != nil {
		return
	}

	for i := range entities.Entities {
		entity := entities.Entities[i]
		results += printEntity(*entity.Category, *entity.Text)
		for j := range entity.Attributes {
			attribute := entity.Attributes[j]
			results += "    Text:   " + *attribute.Text + linebreak
			results += "      Type: " + *attribute.Type + linebreak
		}
	}

	icd10, err := service.InferICD10CM(&icd10i)
	if err != nil {
		return
	}

	for i := range icd10.Entities {
		entity := icd10.Entities[i]
		results += printEntity(*entity.Category, *entity.Text)
		for j := range entity.ICD10CMConcepts {
			concept := entity.ICD10CMConcepts[j]
			results += printConcept(*concept.Code, *concept.Description)
		}
	}

	rxnorm, err := service.InferRxNorm(&rxni)
	if err != nil {
		return
	}

	for i := range rxnorm.Entities {
		entity := rxnorm.Entities[i]
		results += printEntity(*entity.Category, *entity.Text)
		for j := range entity.RxNormConcepts {
			concept := entity.RxNormConcepts[j]
			results += printConcept(*concept.Code, *concept.Description)
		}
	}

	return
}

func printEntity(category string, text string) (result string) {
	return "Category:   " + category + linebreak + "  Text:     " + text + linebreak
}

func printConcept(code string, description string) (result string) {
	result += "    Code:   " + code + linebreak
	result += "      Desc: " + description + linebreak
	return result
}

func errorCheck(err error) {
	if err != nil {
		panic(err)
	}
}
