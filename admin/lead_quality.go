package admin

import (
	"regexp"
	"strings"
)

// MinLeadReviews is the minimum Google Maps review count a business must
// have before it is shown or ingested as a B2B lead.
const MinLeadReviews = 50

var excludedChainTokens = []string{
	"exito",
	"olimpica",
	"d1",
	"ara",
	"carulla",
	"oxxo",
	"falabella",
	"isimo",
	"homecenter",
	"easy",
}

var excludedChainPhrase = []string{
	"justo y bueno",
	"justo bueno",
}

var chainWord = regexp.MustCompile(`\s+`)

// IsExcludedChain reports whether title is a supermarket / retail banner
// that should never enter the lead pool.
func IsExcludedChain(title string) bool {
	key := " " + cityKey(title) + " "
	key = " " + chainWord.ReplaceAllString(strings.TrimSpace(key), " ") + " "

	for _, tok := range excludedChainTokens {
		if strings.Contains(key, " "+tok+" ") {
			return true
		}
	}

	for _, phrase := range excludedChainPhrase {
		if strings.Contains(key, " "+phrase+" ") {
			return true
		}
	}

	return false
}

// QualifiesAsLead reports whether a scraped place should be kept.
func QualifiesAsLead(title string, reviewCount int) bool {
	return reviewCount > MinLeadReviews && !IsExcludedChain(title)
}
