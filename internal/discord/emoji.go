package discord

var appEmoji = map[string]string{
	"little_soggy":     "<:little_soggy:1388591074263568505>",
	"melted_mellow":    "<:melted_mellow:1388591004747043007>",
	"sweet_not_stupid": "<:sweet_not_stupid:1388590903719100436>",
	"youre_valid":      "<:youre_valid:1388590814896066620>",
	"need_a_hug":       "<:need_a_hug:1388590408770261152>",
	"everythings_fine": "<:everythings_fine:1388590355657527306>",
	"hug_me":           "<:hug_me:1388590313999696085>",
	"mellow":           "<:mellow:1388590234568101958>",
}

func mellowEmoji(name string) string {
	if e, ok := appEmoji[name]; ok {
		return e
	}
	return ""
}
