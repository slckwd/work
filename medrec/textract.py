import boto3
import io
import sys

word_filter = lambda block: 'BlockType' in block and 'Text' in block and block['BlockType'] == 'WORD'
linebreak = "\n"

def main():
    text = textract()
    result = comprehend(text)

    file = open("py_results.txt", "w")
    file.write(text + linebreak + linebreak)
    file.write(result)
    file.close()

def textract():
    all_text = ''

    s3_conn = boto3.resource('s3')
    s3_obj = s3_conn.Object('just-this-bucket-you-know', 'test_file.png')
    s3_resp = s3_obj.get()

    stream = io.BytesIO(s3_resp['Body'].read())
    image = stream.getvalue()

    client = boto3.client('textract')
    textract_resp = client.analyze_document(Document={'Bytes': image}, FeatureTypes=['TABLES', 'FORMS'])

    blocks = textract_resp['Blocks']
    for block in filter(word_filter, blocks):
        all_text += ' ' + str(block['Text'])

    return all_text

def comprehend(text):
    results = ''

    client = boto3.client('comprehendmedical')
    entity_resp = client.detect_entities(Text = text)
    icd10_resp = client.infer_icd10_cm(Text = text)
    rxnorm_resp = client.infer_rx_norm(Text = text)

    for entity in entity_resp['Entities']:
        results += print_entity(entity)
        if 'Attributes' in entity:
            for attribute in entity['Attributes']:
                results += "    Text:   " + attribute['Text'] + linebreak
                results += "      Type: " + attribute['Type'] + linebreak

    for entity in icd10_resp['Entities']:
        results += print_entity(entity)
        results += print_concept(entity, 'ICD10CMConcepts')

    for entity in rxnorm_resp['Entities']:
        results += print_entity(entity)
        results += print_concept(entity, 'RxNormConcepts')

    return results

def print_entity(entity):
    return "Category:   " + entity['Category'] + linebreak + "  Text:     " + entity['Text'] + linebreak

def print_concept(entity, concept):
    text = ''
    if concept in entity:
        for code in entity[concept]:
            text += "    Code:   " + code['Code'] + linebreak
            text += "      Desc: " + code['Description'] + linebreak
    return text

if __name__ == "__main__":
    main()