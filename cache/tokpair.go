package cache

// TokenPair : represents one login entry in the cache
type TokenPair struct {
	Auth *JWTok
	Refr *JWTok
}

// MakeMarshalable : token pair needs a object interface{} to call on
func (tp *TokenPair) MakeMarshalable(authSecret, refrSecret string) interface{} {
	result := map[string]string{}
	// All that we need here is converting tokens to
	toks, _ := tp.Auth.ToString(authSecret)
	result["auth"] = string(toks)
	toks, _ = tp.Refr.ToString(refrSecret)
	result["refr"] = string(toks)
	return result
}
