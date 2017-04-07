/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"strconv"
	"encoding/json"
	"time"
	
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

var claimIndexStr = "_claimindex"				//name for the key/value that will store a list of all known SmartClaims


type Customer struct{
	Id string `json:"id"`					//the fieldtags are needed to keep case from bouncing around
	FirstName string `json:"firstName"`
	LastName string `json:"lastName"`
}

type SmartClaim struct{
	Id string `json:"id"`					//the fieldtags are needed to keep case from bouncing around
	CustomerId string `json:"customerId"`
	ClaimDate int64 `json:"claimDate"`
	ClaimAmount string `json:"claimAmount"`
}


func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

func (t *SimpleChaincode) Run(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	return t.Invoke(stub, function, args)
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}
		
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for asset holding")
	}

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval))) 		//making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Invoke isur entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {
		return t.Init(stub, "init", args)
	} else if function == "write" {
		return t.write(stub, args)
	} else if function == "init_claim" {
		return t.init_claim(stub, args)
	}
	fmt.Println("invoke did not find func: " + function)

	return nil, errors.New("Received unknown function invocation: " + function)
}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" { //read a variable
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function)

	return nil, errors.New("Received unknown function query: " + function)
}

// write - invoke function to write key/value pair
func (t *SimpleChaincode) write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, value string
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the key and value to set")
	}

	key = args[0] //rename for funsies
	value = args[1]
	err = stub.PutState(key, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// read - query function to read key/value pair
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")
	}

	key = args[0]
	valAsbytes, err := stub.GetState(key)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil
}


// ============================================================================================================================
// Init Marble - create a new marble, store into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) init_claim(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0       1      2
	//  "100",    "1",  "150"
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	//input sanitation
	fmt.Println("- start init claim")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("3rd argument must be a non-empty string")
	}

	claimId := args[0]
	customerId := args[1]
	amount := args[2]
	claimDate := makeTimestamp()

	//check if marble already exists
	marbleAsBytes, err := stub.GetState(claimId)
	if err != nil {
		return nil, errors.New("Failed to get marble name")
	}
	res := SmartClaim{}
	json.Unmarshal(marbleAsBytes, &res)
	if res.Id == claimId{
		fmt.Println("This claim arleady exists: " + claimId)
		fmt.Println(res);
		return nil, errors.New("This claim arleady exists")				//all stop a claim by this name exists
	}
	
	//build the claim json string manually
	str := `{"id": "` + claimId + `", "customerId": "` + customerId + `", "claimAmount": ` + amount + `, "claimDate": "` + strconv.FormatInt(claimDate, 10) + `"}`
	err = stub.PutState(claimId, []byte(str))									//store marble with id as key
	if err != nil {
		return nil, err
	}
		
	//get the claim index
	claimsAsBytes, err := stub.GetState(claimIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get claim index")
	}
	var claimIndex []string
	json.Unmarshal(claimsAsBytes, &claimIndex)							//un stringify it aka JSON.parse()
	
	//append
	claimIndex = append(claimIndex, claimId)								//add claim id to index list
	fmt.Println("! claim index: ", claimIndex)
	jsonAsBytes, _ := json.Marshal(claimIndex)
	err = stub.PutState(claimIndexStr, jsonAsBytes)						//store id of claim

	fmt.Println("- end init claim")
	return nil, nil
}


// ============================================================================================================================
// Make Timestamp - create a timestamp in ms
// ============================================================================================================================
func makeTimestamp() int64 {
    return time.Now().UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond))
}
