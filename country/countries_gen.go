// +build ignore

/*
Copyright (c) 2018 BlueBoard SAS.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dolmen-go/codegen"
	"golang.org/x/text/unicode/cldr"
)

const template = `// Code generated by "go run countries_gen.go {{.file}}"; DO NOT EDIT.

package country

import "github.com/blueboardio/cldr/currency"

type Info struct {
	Code Code
	Name string
	Currencies []currency.Code
}

// Countries is the list of countries from Unicode CLDR.
//
// Source: {{.file}}
//
// The following codes are removed:
//     QO (duplicate of UM)
//     ZZ (unknown)
var Countries = map[Code]*Info{
{{range .countries -}}
	{{printf "%q: {Code: %q, Name: %q" .Code .Code .Name }}{{ if .Currencies }}, Currencies: []currency.Code{ {{range .Currencies}}"{{.}}",{{end}} }{{end}} },
{{end}}
}

`

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	// Ex: cldr-common-33.0.zip
	cldrArchivePath := os.Args[1]
	zip, err := os.Open(cldrArchivePath)
	if err != nil {
		log.Fatalf("%s: %s", cldrArchivePath, err)
	}

	cldrDecoder := &cldr.Decoder{}
	log.Println("Loading...")
	db, err := cldrDecoder.DecodeZip(zip)
	if err != nil {
		log.Fatalf("%s: %s", cldrArchivePath, err)
	}
	db.SetDraftLevel(cldr.Contributed, false)
	ldml := db.RawLDML("en")
	/*
		ldml, err := db.LDML("en")
		if err != nil {
			log.Fatalf("en: %s", err)
		}
	*/
	supd := db.Supplemental()
	log.Println("Loaded.")

	type country struct {
		Code       string
		Name       string
		Currencies []string
	}
	territories := ldml.LocaleDisplayNames.Territories.Territory
	countries := make(map[string]*country, len(territories))
	for _, ter := range territories {
		// Skip alt="short", alt="variant"
		if len(ter.Alt) > 0 {
			continue
		}
		// Skip continents
		if len(ter.Type) != 2 {
			continue
		}

		// Remove "QO" (in CLDR, but invalid ISO-3166)
		if ter.Type[0] == 'Q' && ter.Type[1] >= 'M' {
			continue
		}
		if ter.Type == "ZZ" {
			continue
		}
		// Remove "X?" except "XK" (special code for Kosovo)
		if ter.Type[0] == 'X' && ter.Type[1] != 'K' {
			continue
		}

		//fmt.Printf("%s %q\n", ter.Type, ter.Data())
		countries[ter.Type] = &country{Code: ter.Type, Name: ter.Data()}
	}

	for _, r := range supd.CurrencyData.Region {

		c := countries[r.Iso3166]
		if c == nil {
			continue
		}

		for _, cu := range r.Currency {
			// We want only tender legal currencies, not "XXX" or Gold
			if cu.Tender == "false" {
				continue
			}
			// We keep only current currencies
			if len(cu.To) > 0 {
				continue
			}

			c.Currencies = append(c.Currencies, cu.Iso4217)
		}

		switch len(c.Currencies) {
		case 1:
		case 0:
			log.Printf("%s: no currencies", c.Code)
		default:
			log.Printf("%s.Currencies: %v", c.Code, c.Currencies)
		}
	}

	fmt.Println(len(countries), "countries.")

	/*
		codegen.MustParse(template).Template.Execute(os.Stdout, map[string]interface{}{
			"file":      filepath.Base(cldrArchivePath),
			"countries": countries,
		})
	*/

	err = codegen.MustParse(template).CreateFile("countries.go", map[string]interface{}{
		"file":      filepath.Base(cldrArchivePath),
		"countries": countries,
	})
	if err != nil {
		log.Fatal(err)
	}
}
