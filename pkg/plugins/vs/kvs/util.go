package kvs

import "io/ioutil"

//get resource yaml
func GetResourceYaml(filePath string) []byte {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err.Error())
	}
	return bytes
}
