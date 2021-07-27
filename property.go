package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

//creation of property transfer smart contract
type PropertyTransferSmartContract struct {
	contractapi.Contract
}

//Property struct stored on blockchain
type Property struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Area         int    `json:"area"`
	OwnerName    string `json:"ownerName"`
	Value        int    `json:"value"`
	BitcoinValue float64
}

//Bitcoin API call JSON structs
type OuterRate struct {
	Holder Rate `json:"Realtime Currency Exchange Rate"`
}

type Rate struct {
	FromCurCode string `json:"1. From_Currency Code"`
	FromCurName string `json:"2. From_Currency Name"`
	ToCurCode   string `json:"3. To_Currency Code"`
	ToCurName   string `json:"4. To_Currency Name"`
	ExRate      string `json:"5. Exchange Rate"`
	LastRefresh string `json:"6. Last Refreshed"`
	Zone        string `json:"7. Time Zone"`
	Bid         string `json:"8. Bid Price"`
	Ask         string `json:"9. Ask Price"`
}

//ADD A NEW PROPERTY TO THE LEDGER
func (pc *PropertyTransferSmartContract) AddProperty(ctx contractapi.TransactionContextInterface,
	id string, name string, area int, ownerName string, value int) error {
	//Check if ID already exists
	propertyJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("failed to read the data from world state")
	}
	if propertyJSON != nil {
		return fmt.Errorf("fhe property %s already exists", id)
	}

	//Create object to place on blockchain
	prop := Property{
		ID:           id,
		Name:         name,
		Area:         area,
		OwnerName:    ownerName,
		Value:        value,
		BitcoinValue: 0.0,
	}

	//BITCOIN exchange rate API call to get current rate:
	response, err := http.Get("https://www.alphavantage.co/query?function=CURRENCY_EXCHANGE_RATE&from_currency=BTC&to_currency=USD&apikey=/*apikeyhere*/")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	var result OuterRate
	_ = decoder.Decode(&result)

	//BITCOIN Value calculation:
	btcExRate, _ := strconv.ParseFloat(result.Holder.ExRate, 64)
	prop.BitcoinValue = float64(value) / btcExRate

	//Final JSON package
	propertyBytes, err := json.Marshal(prop) //create json encoding
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(id, propertyBytes) //pass json to api

}

//QUERY ALL EXISTING PROPERTIES
func (pc *PropertyTransferSmartContract) QueryAllProperties(ctx contractapi.TransactionContextInterface) ([]*Property, error) {
	//Range with blank values will return all values
	propertyIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer propertyIterator.Close()
	//return value definition
	var properties []*Property
	for propertyIterator.HasNext() {
		//iterate through each property that was returned
		propertyResponse, err := propertyIterator.Next()
		if err != nil {
			return nil, err
		}
		//convert each from JSON and append to return val
		var property *Property
		err = json.Unmarshal(propertyResponse.Value, &property)
		if err != nil {
			return nil, err
		}
		properties = append(properties, property)
	}
	return properties, nil
}

//QUERY PROPERTIES BY ID
func (pc *PropertyTransferSmartContract) QueryPropertyByID(ctx contractapi.TransactionContextInterface, id string) (*Property, error) {
	//get property by ID
	propertyJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read the data from world state")
	}
	if propertyJSON == nil {
		return nil, fmt.Errorf("the property %s does not exist", id)
	}
	//convert json to golang obj
	var property *Property
	err = json.Unmarshal(propertyJSON, &property)
	if err != nil {
		return nil, err
	}
	return property, nil
}

//TRANSFER PROPERTY OWNERSHIP
func (pc *PropertyTransferSmartContract) TransferProperty(ctx contractapi.TransactionContextInterface,
	id string, newOwner string) error {
	//get current property information
	property, err := pc.QueryPropertyByID(ctx, id)
	if err != nil {
		return err
	}
	//update property ownership
	property.OwnerName = newOwner
	propertyJSON, err := json.Marshal(property)
	if err != nil {
		return err
	}
	//return updated property ownership
	return ctx.GetStub().PutState(id, propertyJSON)
}

func main() {
	//new instance of smart contract struct
	propTransferSmartContract := new(PropertyTransferSmartContract)
	//Add chain code
	cc, err := contractapi.NewChaincode(propTransferSmartContract)
	if err != nil {
		panic(err.Error())
	}
	//Init chain code
	if err := cc.Start(); err != nil {
		panic(err.Error())
	}
}
