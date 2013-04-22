package models

type NestLoginURLs struct{
	Transport_url, Rubyapi_url, Weather_url, Log_upload_url, Support_url string
}

type UserInfo struct{
	Email string
	Hash string
}

type NestLogin struct{
	Is_superuser bool 
	Urls NestLoginURLs
	Access_token string
	Userid string
	Expires_in string
	Email string
	User string
}

type Keys struct{
	Keys []Key `json:"keys"`
}

type Key struct{
	Key string `json:"key"`
	Version uint64 `json:"version"`
	Timestamp uint64 `json:"timestamp"`
}


type Cycle struct{
	Start uint64 `json:"start"`
	Duration uint64 `json:"duration"`
	Type int `json:"type"`
}

type Event struct{
	Start uint64 `json:"start"`
	End uint64 `json:"end"`
	Type uint64 `json:"type"`
	Continuation bool `json:"continuation"`
	Heat_Temp float32 `json:"heat_temp"`
	Touched_By int `json:"touched_by"`
	Touched_When int `json:"touched_when"`
	Touched_Timezone_Offset int `json:"touched_timezone_offset"`
	Touched_Id string `json:"touched_id"`
	Touched_Source string `json:"touched_source"`
}

type Day struct{
	Cycles []Cycle `json:"cycles"`
	Events []Event `json:"events"`
	Day string `json:"day"`
	Device_timezone_offset int `json:"device_timezone_offset"`
	Total_heating_time int `json:"total_heating_time"`
	Total_cooling_time int `json:"total_cooling_time"`
	Total_fan_cooling_time int `json:"total_fan_cooling_time"`
	Total_humidifier_time int `json:"total_humidifier_time"`
	Total_dehumidifier_time int `json:"total_dehumidifier_time"`
	Leafs int `json:"leafs"`
	Recent_avg_used int `json:"recent_avg_used"`
	Usage_over_avg int `json:"usage_over_age"`
	Whodunit int `json:"whodunit"`
}

type DayDataStore struct{
	Email string
	Day []byte
	Date string
	Leafs int
	Total_heating_time int
	Total_cooling_time int
	Total_fan_cooling_time int
	Total_humidifier_time int
	Total_dehumidifier_time int
	Recent_avg_used int
	Usage_over_avg int
	Whodunit int
}

type Payload struct{
	Days []Day `json:"days"`
}

type Energy struct{
	Status uint16 `json:"status"`
	Payload Payload `json:"payload"`
}