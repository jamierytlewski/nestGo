package main 

import _ "appengine/remote_api"

import (
	"net/http"
	"net/url"
	"fmt"
	"log"
	"io/ioutil"
	"html/template"
	"encoding/json"
	"strings"
	"calculations"
	"models"
	"nestdatastore"
	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"time"
	"encoding/csv"
	"strconv"
	)

type Page struct{
	Title string
	Body []byte
}

func root(w http.ResponseWriter, r *http.Request) {
	/* Gets the cookie and if the cookie is not set then
	redirect to the login page */
	_, err := r.Cookie("email")
	if err != nil{
		http.Redirect(w, r, "/login", 301)
		return
	}

    err = indexTmpl.ExecuteTemplate(w, "base.html", nil)
    if err != nil{
    	http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

func api_energy(w http.ResponseWriter, r *http.Request){
	cookie, err := r.Cookie("email")
	if err != nil{
		http.Redirect(w, r, "/login", 301)
		return
	}
    // Gets the last 30 days
    energy := nestdatastore.GetValuesFromDataStore(30, cookie.Value, w, r)
    dayArray := nestdatastore.GetDayArray(energy)
   	dayJson, _ := json.Marshal(dayArray)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "[%s]", dayJson)
}

func login(w http.ResponseWriter, r *http.Request){
	err := loginTmpl.ExecuteTemplate(w, "base.html", nil)
    if err != nil{
    	http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loginSubmit(w http.ResponseWriter, r *http.Request){
	/* Gets the values from the form */
	email := r.FormValue("email")
	password := r.FormValue("password")

	/* Sets the Cookie */
	expire := time.Now().AddDate(1, 0, 0)
	cookie := http.Cookie{Name:"email", Value:email, Expires:expire, MaxAge:86400}
	cookie.Name = "email"
	http.SetCookie(w, &cookie)

	c := appengine.NewContext(r)
	client := urlfetch.Client(c)
	/* This is doing the initial call */
	resp, err := client.PostForm("https://home.nest.com/user/login", url.Values{"username":{email}, "password": {password}})

	if err != nil{
		fmt.Printf("Error!!")
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	
	var nestLogin models.NestLogin
	errJson := json.Unmarshal(body, &nestLogin)
	if errJson != nil{
		log.Fatal(errJson)
	}

	s := `{"keys":[{"key":"` + nestLogin.User + `","version":-843004088,"timestamp":1356287364000},` + 
		`{"key":"user_alert_dialog.188262","version":-1320746469,"timestamp":1356287465000},`+
		`{"key":"user_settings.188262","version":1496039529,"timestamp":1356287392615},` +
		`{"key":"device.02AA01AB44120G73","version":1348169463,"timestamp":1360446485000},`+ 
		`{"key":"shared.02AA01AB44120G73","version":1309307191,"timestamp":1360454917000},`+
		`{"key":"schedule.02AA01AB44120G73","version":-79023839,"timestamp":1360387822000},`+
		`{"key":"track.02AA01AB44120G73","version":583994169,"timestamp":1360458455726},`+
		`{"key":"energy_latest.02AA01AB44120G73"}]}`
	req, err := http.NewRequest("POST", nestLogin.Urls.Transport_url + "/v2/subscribe", strings.NewReader(s))
	req.ContentLength = int64(len(s)) 
	req.Header.Add("Authorization", "Basic " + nestLogin.Access_token)
	req.Header.Add("X-nl-protocol-version", "1")
	req.Header.Add("X-nl-client-timestamp", "1360458454279")
	req.Header.Add("X-nl-session-id", "136045845235226189601.188525558")
	req.Header.Add("X-nl-subscribe-timeout", "60")
	req.Header.Add("User-Agent", "Nest/3.0.15 (iOS) os=6.0 platform=iPad3,1")
	resp1, err := client.Do(req)

    if err != nil{
            fmt.Printf("Error!!")
            log.Fatal(err)
    }

    body1, err := ioutil.ReadAll(resp1.Body)

    energy10 := nestdatastore.GetValuesFromDataStoreForImport(email, w, r)
	dayArray := nestdatastore.GetDayArray(energy10)

	var energy models.Payload
	errJson = json.Unmarshal(body1, &energy)
	if errJson != nil{
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	for _, value := range energy.Days{
		_, ok := dayArray[value.Day]
		if ok == true{
			continue
		}
		day, err1 := json.Marshal(value)
		if err1 != nil{
			fmt.Println("error:", err1)
		}
		c := appengine.NewContext(r)
		g := models.DayDataStore{
			Email: email,
			Day: day,
			Date: value.Day,
			Total_heating_time: value.Total_heating_time,
			Total_cooling_time: value.Total_cooling_time,
			Total_fan_cooling_time: value.Total_fan_cooling_time,
			Total_humidifier_time: value.Total_humidifier_time,
			Total_dehumidifier_time: value.Total_dehumidifier_time,
			Leafs: value.Leafs,
			Whodunit: value.Whodunit,
			Recent_avg_used: value.Recent_avg_used,
			Usage_over_avg: value.Usage_over_avg,
		}
		_, err := datastore.Put(c, datastore.NewIncompleteKey(c, "DayDataStore", nil), &g)
	    if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	        return
	    }
	}
	
	http.Redirect(w, r, "/", 301)
	return
}


var funcMap = template.FuncMap{
		// The name "title" is what the function will be called in the template text.
		"ctof": calculations.CtoF,
	}
var indexTmpl = template.Must(template.New("body").Funcs(funcMap).ParseFiles("tmpl/index.html", "tmpl/base.html"))
var loginTmpl = template.Must(template.New("body").Funcs(funcMap).ParseFiles("tmpl/login.html", "tmpl/base.html"))
func sign(w http.ResponseWriter, r *http.Request){
	energy10 := nestdatastore.GetValuesFromDataStore(10, "jamier.net@gmail.com", w, r)
	dayArray := nestdatastore.GetDayArray(energy10)

	/*for key, _ := range dayArray{
		fmt.Fprintln(w, "%s", key)
	}*/

	energyString := `{"status":200,"headers":{"X-nl-skv-key":"energy_latest.02AA01AB44120G73","X-nl-skv-version":1,"X-nl-skv-timestamp":1363504012193,"X-nl-service-timestamp":1363569418777},"payload":{"days":[{"day":"2013-03-07","device_timezone_offset":-18000,"total_heating_time":26190,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":25105,"usage_over_avg":1085,"cycles":[{"start":0,"duration":628,"type":1},{"start":5842,"duration":1139,"type":1},{"start":12195,"duration":1138,"type":1},{"start":17140,"duration":1258,"type":1},{"start":19800,"duration":8303,"type":1},{"start":53837,"duration":4393,"type":1},{"start":60566,"duration":4772,"type":1},{"start":67528,"duration":1256,"type":1},{"start":71089,"duration":1227,"type":1},{"start":74141,"duration":1047,"type":1},{"start":77701,"duration":1499,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":44815,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":44816,"end":50494,"type":4,"heat_temp":10.0},{"start":50495,"end":53835,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":53836,"end":53836,"type":0,"heat_temp":15.362,"touched_by":2,"touched_when":1362686236,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":53837,"end":53837,"type":0,"heat_temp":16.387,"touched_by":2,"touched_when":1362686237,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":53838,"end":53838,"type":0,"heat_temp":17.085,"touched_by":2,"touched_when":1362686238,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":53839,"end":53839,"type":0,"heat_temp":17.258,"touched_by":2,"touched_when":1362686239,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":53840,"end":61199,"type":0,"heat_temp":17.566,"touched_by":2,"touched_when":1362686240,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-08","device_timezone_offset":-18000,"total_heating_time":20010,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":1,"recent_avg_used":25865,"usage_over_avg":-5855,"cycles":[{"start":16003,"duration":1199,"type":1},{"start":19800,"duration":8783,"type":1},{"start":57447,"duration":7004,"type":1},{"start":66793,"duration":1009,"type":1},{"start":72918,"duration":779,"type":1},{"start":76749,"duration":1256,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":57444,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":57445,"end":57445,"type":0,"heat_temp":12.05,"touched_by":2,"touched_when":1362776245,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":57446,"end":57446,"type":0,"heat_temp":16.086,"touched_by":2,"touched_when":1362776246,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":57447,"end":57447,"type":0,"heat_temp":18.84,"touched_by":2,"touched_when":1362776247,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":57448,"end":61199,"type":0,"heat_temp":19.385,"touched_by":2,"touched_when":1362776248,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-09","device_timezone_offset":-18000,"total_heating_time":22500,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":24335,"usage_over_avg":-1835,"cycles":[{"start":14838,"duration":1198,"type":1},{"start":19122,"duration":1258,"type":1},{"start":23106,"duration":1258,"type":1},{"start":26100,"duration":7129,"type":1},{"start":34725,"duration":1496,"type":1},{"start":37987,"duration":1256,"type":1},{"start":56124,"duration":735,"type":1},{"start":62930,"duration":6198,"type":1},{"start":70954,"duration":1227,"type":1},{"start":74964,"duration":1197,"type":1}],"events":[{"start":0,"end":26099,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":26100,"end":39599,"type":0,"heat_temp":19.445,"touched_by":4,"touched_when":1359863500,"touched_timezone_offset":-18000},{"start":39600,"end":52626,"type":0,"heat_temp":16.667,"touched_by":1,"touched_when":1360560621,"touched_timezone_offset":-18000},{"start":52627,"end":54081,"type":4,"heat_temp":10.0},{"start":54082,"end":56335,"type":0,"heat_temp":16.667,"touched_by":1,"touched_when":1360560621,"touched_timezone_offset":-18000},{"start":56336,"end":56336,"type":0,"heat_temp":17.442,"touched_by":2,"touched_when":1362861536,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":56337,"end":56337,"type":0,"heat_temp":19.005,"touched_by":2,"touched_when":1362861537,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":56338,"end":56338,"type":0,"heat_temp":19.383,"touched_by":2,"touched_when":1362861538,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":56339,"end":79199,"type":0,"heat_temp":19.37,"touched_by":2,"touched_when":1362861539,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":4,"touched_when":1359490018,"touched_timezone_offset":-18000}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-10","device_timezone_offset":-14400,"total_heating_time":13020,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":1,"recent_avg_used":23160,"usage_over_avg":-10140,"cycles":[{"start":19791,"duration":1135,"type":1,"timezone_offset":-18000},{"start":22500,"duration":6719,"type":1,"timezone_offset":-18000},{"start":31582,"duration":1107,"type":1,"timezone_offset":-18000},{"start":35860,"duration":140,"type":1,"timezone_offset":-18000},{"start":41400,"duration":1639,"type":1,"timezone_offset":-18000},{"start":49471,"duration":1136,"type":1,"timezone_offset":-18000},{"start":69753,"duration":1227,"type":1,"timezone_offset":-18000}],"events":[{"start":0,"end":22499,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":4,"touched_when":1359490018,"touched_timezone_offset":-18000,"timezone_offset":-18000},{"start":22500,"end":35999,"type":0,"heat_temp":19.445,"touched_by":4,"touched_when":1359863503,"touched_timezone_offset":-18000,"timezone_offset":-18000},{"start":36000,"end":41399,"type":0,"heat_temp":16.667,"touched_by":3,"touched_when":1360512736,"touched_timezone_offset":-18000,"touched_source":"thermozilla","timezone_offset":-18000},{"start":41400,"end":75599,"type":0,"heat_temp":19.332,"touched_by":2,"touched_when":1362332519,"touched_timezone_offset":-18000,"touched_source":"dial","timezone_offset":-18000},{"start":75600,"end":86399,"type":0,"heat_temp":15.556,"touched_by":4,"touched_when":1359491030,"touched_timezone_offset":-18000,"timezone_offset":-18000}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-11","device_timezone_offset":-14400,"total_heating_time":10800,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":1,"recent_avg_used":20900,"usage_over_avg":-10100,"cycles":[{"start":19800,"duration":4059,"type":1},{"start":54329,"duration":2723,"type":1},{"start":61200,"duration":1118,"type":1},{"start":68153,"duration":1227,"type":1},{"start":74080,"duration":808,"type":1},{"start":78091,"duration":898,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":4,"touched_when":1359491030,"touched_timezone_offset":-18000},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":4,"touched_when":1359509821,"touched_timezone_offset":-18000},{"start":28800,"end":54326,"type":0,"heat_temp":10.0,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000},{"start":54327,"end":54327,"type":0,"heat_temp":10.653,"touched_by":2,"touched_when":1363028727,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":54328,"end":54328,"type":0,"heat_temp":17.437,"touched_by":2,"touched_when":1363028728,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":54329,"end":54329,"type":0,"heat_temp":18.898,"touched_by":2,"touched_when":1363028729,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":54330,"end":54330,"type":0,"heat_temp":19.276,"touched_by":2,"touched_when":1363028730,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":54331,"end":61199,"type":0,"heat_temp":19.321,"touched_by":2,"touched_when":1363028730,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766946,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-12","device_timezone_offset":-14400,"total_heating_time":19590,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":18470,"usage_over_avg":1120,"cycles":[{"start":19800,"duration":7070,"type":1},{"start":41308,"duration":3787,"type":1},{"start":50571,"duration":898,"type":1},{"start":54521,"duration":1167,"type":1},{"start":58739,"duration":1437,"type":1},{"start":61200,"duration":2058,"type":1},{"start":67656,"duration":1286,"type":1},{"start":72712,"duration":1077,"type":1},{"start":76092,"duration":838,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":41305,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":41306,"end":41306,"type":0,"heat_temp":11.326,"touched_by":2,"touched_when":1363102106,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":41307,"end":41307,"type":0,"heat_temp":16.957,"touched_by":2,"touched_when":1363102107,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":41308,"end":41308,"type":0,"heat_temp":18.526,"touched_by":2,"touched_when":1363102108,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":41309,"end":41310,"type":0,"heat_temp":18.571,"touched_by":2,"touched_when":1363102109,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":41311,"end":61199,"type":0,"heat_temp":18.712,"touched_by":2,"touched_when":1363102111,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-13","device_timezone_offset":-14400,"total_heating_time":26010,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":0,"recent_avg_used":18685,"usage_over_avg":7325,"cycles":[{"start":16038,"duration":1198,"type":1},{"start":19800,"duration":8669,"type":1},{"start":43746,"duration":6491,"type":1},{"start":52512,"duration":1078,"type":1},{"start":57032,"duration":1077,"type":1},{"start":60293,"duration":1287,"type":1},{"start":64152,"duration":1765,"type":1},{"start":67653,"duration":1705,"type":1},{"start":74695,"duration":2803,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":43743,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":43744,"end":43744,"type":0,"heat_temp":11.095,"touched_by":2,"touched_when":1363190944,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":43745,"end":43745,"type":0,"heat_temp":16.156,"touched_by":2,"touched_when":1363190945,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":43746,"end":43746,"type":0,"heat_temp":18.661,"touched_by":2,"touched_when":1363190946,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":43747,"end":61199,"type":0,"heat_temp":19.366,"touched_by":2,"touched_when":1363190947,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":61200,"end":70817,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":70818,"end":74694,"type":4,"heat_temp":10.0},{"start":74695,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-14","device_timezone_offset":-14400,"total_heating_time":22110,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":18655,"usage_over_avg":3455,"cycles":[{"start":11479,"duration":1257,"type":1},{"start":15880,"duration":1168,"type":1},{"start":19800,"duration":8117,"type":1},{"start":61200,"duration":8823,"type":1},{"start":75423,"duration":2767,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":45026,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":45027,"end":50228,"type":4,"heat_temp":10.0},{"start":50229,"end":61199,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":61200,"end":71848,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":71849,"end":75421,"type":4,"heat_temp":10.0},{"start":75422,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-15","device_timezone_offset":-14400,"total_heating_time":23370,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":19005,"usage_over_avg":4365,"cycles":[{"start":18342,"duration":1108,"type":1},{"start":19800,"duration":7525,"type":1},{"start":34569,"duration":3590,"type":1},{"start":40343,"duration":1196,"type":1},{"start":46755,"duration":1815,"type":1},{"start":50815,"duration":1256,"type":1},{"start":55483,"duration":1376,"type":1},{"start":60809,"duration":1257,"type":1},{"start":66702,"duration":1317,"type":1},{"start":71610,"duration":1525,"type":1},{"start":75739,"duration":1406,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":33083,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":33084,"end":34555,"type":4,"heat_temp":10.0},{"start":34556,"end":34565,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":34566,"end":34566,"type":0,"heat_temp":14.254,"touched_by":2,"touched_when":1363354566,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":34567,"end":34567,"type":0,"heat_temp":16.316,"touched_by":2,"touched_when":1363354567,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":34568,"end":34568,"type":0,"heat_temp":17.995,"touched_by":2,"touched_when":1363354568,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":34569,"end":34569,"type":0,"heat_temp":18.949,"touched_by":2,"touched_when":1363354569,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":34570,"end":44740,"type":0,"heat_temp":19.391,"touched_by":2,"touched_when":1363354570,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":44741,"end":46754,"type":4,"heat_temp":10.0},{"start":46755,"end":61199,"type":0,"heat_temp":19.391,"touched_by":2,"touched_when":1363354570,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-16","device_timezone_offset":-14400,"total_heating_time":19980,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":19150,"usage_over_avg":830,"cycles":[{"start":14996,"duration":1198,"type":1},{"start":20448,"duration":1198,"type":1},{"start":25391,"duration":7666,"type":1},{"start":35272,"duration":1346,"type":1},{"start":58169,"duration":1108,"type":1},{"start":63798,"duration":1019,"type":1},{"start":68950,"duration":1137,"type":1},{"start":71844,"duration":5221,"type":1},{"start":79129,"duration":71,"type":1}],"events":[{"start":0,"end":26099,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":26100,"end":39599,"type":0,"heat_temp":19.445,"touched_by":4,"touched_when":1359863500,"touched_timezone_offset":-18000},{"start":39600,"end":71842,"type":0,"heat_temp":16.667,"touched_by":1,"touched_when":1360560621,"touched_timezone_offset":-18000},{"start":71843,"end":71843,"type":0,"heat_temp":16.679,"touched_by":2,"touched_when":1363478243,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":71844,"end":71844,"type":0,"heat_temp":19.658,"touched_by":2,"touched_when":1363478244,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":71845,"end":71845,"type":0,"heat_temp":19.357,"touched_by":2,"touched_when":1363478245,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":71846,"end":79199,"type":0,"heat_temp":19.479,"touched_by":2,"touched_when":1363478246,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":4,"touched_when":1359490018,"touched_timezone_offset":-18000}],"system_capabilities":2561,"incomplete_fields":0}],"recent_max_used":35760}}`
	energyString = `{"status":200,"headers":{"X-nl-skv-key":"energy_latest.02AA01AB44120G73","X-nl-skv-version":1,"X-nl-skv-timestamp":1363848801805,"X-nl-service-timestamp":1363921507144},"payload":{"days":[{"day":"2013-03-11","device_timezone_offset":-14400,"total_heating_time":10800,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":1,"recent_avg_used":20900,"usage_over_avg":-10100,"cycles":[{"start":19800,"duration":4059,"type":1},{"start":54329,"duration":2723,"type":1},{"start":61200,"duration":1118,"type":1},{"start":68153,"duration":1227,"type":1},{"start":74080,"duration":808,"type":1},{"start":78091,"duration":898,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":4,"touched_when":1359491030,"touched_timezone_offset":-18000},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":4,"touched_when":1359509821,"touched_timezone_offset":-18000},{"start":28800,"end":54326,"type":0,"heat_temp":10.0,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000},{"start":54327,"end":54327,"type":0,"heat_temp":10.653,"touched_by":2,"touched_when":1363028727,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":54328,"end":54328,"type":0,"heat_temp":17.437,"touched_by":2,"touched_when":1363028728,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":54329,"end":54329,"type":0,"heat_temp":18.898,"touched_by":2,"touched_when":1363028729,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":54330,"end":54330,"type":0,"heat_temp":19.276,"touched_by":2,"touched_when":1363028730,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":54331,"end":61199,"type":0,"heat_temp":19.321,"touched_by":2,"touched_when":1363028730,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766946,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-12","device_timezone_offset":-14400,"total_heating_time":19590,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":18470,"usage_over_avg":1120,"cycles":[{"start":19800,"duration":7070,"type":1},{"start":41308,"duration":3787,"type":1},{"start":50571,"duration":898,"type":1},{"start":54521,"duration":1167,"type":1},{"start":58739,"duration":1437,"type":1},{"start":61200,"duration":2058,"type":1},{"start":67656,"duration":1286,"type":1},{"start":72712,"duration":1077,"type":1},{"start":76092,"duration":838,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":41305,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":41306,"end":41306,"type":0,"heat_temp":11.326,"touched_by":2,"touched_when":1363102106,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":41307,"end":41307,"type":0,"heat_temp":16.957,"touched_by":2,"touched_when":1363102107,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":41308,"end":41308,"type":0,"heat_temp":18.526,"touched_by":2,"touched_when":1363102108,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":41309,"end":41310,"type":0,"heat_temp":18.571,"touched_by":2,"touched_when":1363102109,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":41311,"end":61199,"type":0,"heat_temp":18.712,"touched_by":2,"touched_when":1363102111,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-13","device_timezone_offset":-14400,"total_heating_time":26010,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":0,"recent_avg_used":18685,"usage_over_avg":7325,"cycles":[{"start":16038,"duration":1198,"type":1},{"start":19800,"duration":8669,"type":1},{"start":43746,"duration":6491,"type":1},{"start":52512,"duration":1078,"type":1},{"start":57032,"duration":1077,"type":1},{"start":60293,"duration":1287,"type":1},{"start":64152,"duration":1765,"type":1},{"start":67653,"duration":1705,"type":1},{"start":74695,"duration":2803,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":43743,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":43744,"end":43744,"type":0,"heat_temp":11.095,"touched_by":2,"touched_when":1363190944,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":43745,"end":43745,"type":0,"heat_temp":16.156,"touched_by":2,"touched_when":1363190945,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":43746,"end":43746,"type":0,"heat_temp":18.661,"touched_by":2,"touched_when":1363190946,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":43747,"end":61199,"type":0,"heat_temp":19.366,"touched_by":2,"touched_when":1363190947,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":61200,"end":70817,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":70818,"end":74694,"type":4,"heat_temp":10.0},{"start":74695,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-14","device_timezone_offset":-14400,"total_heating_time":22110,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":18655,"usage_over_avg":3455,"cycles":[{"start":11479,"duration":1257,"type":1},{"start":15880,"duration":1168,"type":1},{"start":19800,"duration":8117,"type":1},{"start":61200,"duration":8823,"type":1},{"start":75423,"duration":2767,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":45026,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":45027,"end":50228,"type":4,"heat_temp":10.0},{"start":50229,"end":61199,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":61200,"end":71848,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":71849,"end":75421,"type":4,"heat_temp":10.0},{"start":75422,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-15","device_timezone_offset":-14400,"total_heating_time":23370,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":19005,"usage_over_avg":4365,"cycles":[{"start":18342,"duration":1108,"type":1},{"start":19800,"duration":7525,"type":1},{"start":34569,"duration":3590,"type":1},{"start":40343,"duration":1196,"type":1},{"start":46755,"duration":1815,"type":1},{"start":50815,"duration":1256,"type":1},{"start":55483,"duration":1376,"type":1},{"start":60809,"duration":1257,"type":1},{"start":66702,"duration":1317,"type":1},{"start":71610,"duration":1525,"type":1},{"start":75739,"duration":1406,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766965,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":33083,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":33084,"end":34555,"type":4,"heat_temp":10.0},{"start":34556,"end":34565,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":34566,"end":34566,"type":0,"heat_temp":14.254,"touched_by":2,"touched_when":1363354566,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":34567,"end":34567,"type":0,"heat_temp":16.316,"touched_by":2,"touched_when":1363354567,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":34568,"end":34568,"type":0,"heat_temp":17.995,"touched_by":2,"touched_when":1363354568,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":34569,"end":34569,"type":0,"heat_temp":18.949,"touched_by":2,"touched_when":1363354569,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":34570,"end":44740,"type":0,"heat_temp":19.391,"touched_by":2,"touched_when":1363354570,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":44741,"end":46754,"type":4,"heat_temp":10.0},{"start":46755,"end":61199,"type":0,"heat_temp":19.391,"touched_by":2,"touched_when":1363354570,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-16","device_timezone_offset":-14400,"total_heating_time":19980,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":19150,"usage_over_avg":830,"cycles":[{"start":14996,"duration":1198,"type":1},{"start":20448,"duration":1198,"type":1},{"start":25391,"duration":7666,"type":1},{"start":35272,"duration":1346,"type":1},{"start":58169,"duration":1108,"type":1},{"start":63798,"duration":1019,"type":1},{"start":68950,"duration":1137,"type":1},{"start":71844,"duration":5221,"type":1},{"start":79129,"duration":71,"type":1}],"events":[{"start":0,"end":26099,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766967,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":26100,"end":39599,"type":0,"heat_temp":19.445,"touched_by":4,"touched_when":1359863500,"touched_timezone_offset":-18000},{"start":39600,"end":71842,"type":0,"heat_temp":16.667,"touched_by":1,"touched_when":1360560621,"touched_timezone_offset":-18000},{"start":71843,"end":71843,"type":0,"heat_temp":16.679,"touched_by":2,"touched_when":1363478243,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":71844,"end":71844,"type":0,"heat_temp":19.658,"touched_by":2,"touched_when":1363478244,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":71845,"end":71845,"type":0,"heat_temp":19.357,"touched_by":2,"touched_when":1363478245,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":71846,"end":79199,"type":0,"heat_temp":19.479,"touched_by":2,"touched_when":1363478246,"touched_timezone_offset":-14400,"touched_source":"dial"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":4,"touched_when":1359490018,"touched_timezone_offset":-18000}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-17","device_timezone_offset":-14400,"total_heating_time":24180,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":1,"recent_avg_used":20310,"usage_over_avg":3870,"cycles":[{"start":11797,"duration":1258,"type":1},{"start":16469,"duration":1228,"type":1},{"start":20962,"duration":1288,"type":1},{"start":25425,"duration":8021,"type":1},{"start":60302,"duration":8188,"type":1},{"start":72679,"duration":2714,"type":1},{"start":77218,"duration":1466,"type":1}],"events":[{"start":0,"end":26099,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":4,"touched_when":1359490018,"touched_timezone_offset":-18000},{"start":26100,"end":33444,"type":0,"heat_temp":19.445,"touched_by":4,"touched_when":1359863503,"touched_timezone_offset":-18000},{"start":33445,"end":60300,"type":3,"heat_temp":10.0},{"start":60301,"end":68489,"type":0,"heat_temp":19.332,"touched_by":2,"touched_when":1362332519,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":68490,"end":72678,"type":4,"heat_temp":10.0},{"start":72679,"end":79199,"type":0,"heat_temp":19.332,"touched_by":2,"touched_when":1362332519,"touched_timezone_offset":-18000,"touched_source":"dial"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":4,"touched_when":1359491030,"touched_timezone_offset":-18000}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-18","device_timezone_offset":-14400,"total_heating_time":25080,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":22540,"usage_over_avg":2540,"cycles":[{"start":8568,"duration":1257,"type":1},{"start":12699,"duration":1349,"type":1},{"start":16923,"duration":1288,"type":1},{"start":19800,"duration":8681,"type":1},{"start":61200,"duration":10011,"type":1},{"start":72916,"duration":1227,"type":1},{"start":75878,"duration":1136,"type":1},{"start":79079,"duration":121,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":4,"touched_when":1359491030,"touched_timezone_offset":-18000},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":4,"touched_when":1359509821,"touched_timezone_offset":-18000},{"start":28800,"end":46897,"type":0,"heat_temp":10.0,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000},{"start":46898,"end":60828,"type":4,"heat_temp":10.0},{"start":60829,"end":61199,"type":0,"heat_temp":10.0,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000},{"start":61200,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766946,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-19","device_timezone_offset":-14400,"total_heating_time":24180,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":-1,"recent_avg_used":23455,"usage_over_avg":725,"cycles":[{"start":12615,"duration":1198,"type":1},{"start":17018,"duration":1228,"type":1},{"start":19800,"duration":8654,"type":1},{"start":61201,"duration":11879,"type":1},{"start":75384,"duration":1227,"type":1}],"events":[{"start":0,"end":19799,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":4,"touched_when":1359489979,"touched_timezone_offset":-18000},{"start":19800,"end":28800,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28801,"end":42107,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":42108,"end":61200,"type":4,"heat_temp":10.0},{"start":61201,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0},{"day":"2013-03-20","device_timezone_offset":-14400,"total_heating_time":25980,"total_cooling_time":0,"total_fan_cooling_time":0,"total_humidifier_time":0,"total_dehumidifier_time":0,"leafs":1,"whodunit":1,"recent_avg_used":23150,"usage_over_avg":2830,"cycles":[{"start":9760,"duration":1301,"type":1},{"start":13822,"duration":1377,"type":1},{"start":17896,"duration":1348,"type":1},{"start":19800,"duration":8606,"type":1},{"start":61200,"duration":10304,"type":1},{"start":71770,"duration":3329,"type":1},{"start":78151,"duration":1049,"type":1}],"events":[{"start":0,"end":11064,"type":0,"continuation":true,"heat_temp":15.556,"touched_by":6,"touched_when":1359766960,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":11065,"end":19799,"type":0,"heat_temp":15.556,"touched_by":3,"touched_when":1363763061,"touched_timezone_offset":-14400,"touched_source":"thermozilla"},{"start":19800,"end":28799,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":28800,"end":48102,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":48103,"end":50720,"type":4,"heat_temp":10.0},{"start":50721,"end":58516,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":58517,"end":59337,"type":4,"heat_temp":10.0},{"start":59338,"end":61199,"type":0,"heat_temp":10.0,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":61200,"end":71503,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":71504,"end":71769,"type":4,"heat_temp":10.0},{"start":71770,"end":79199,"type":0,"heat_temp":19.445,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"},{"start":79200,"end":86399,"type":0,"heat_temp":15.556,"touched_by":6,"touched_when":1359766963,"touched_timezone_offset":-18000,"touched_id":"Jamie R Rytlewski\u2019s iPhone"}],"system_capabilities":2561,"incomplete_fields":0}],"recent_max_used":35760}}`
	var energyByte = []byte(energyString)
	//copy(energyByte, energyString)
	var energy models.Energy
	errJson := json.Unmarshal(energyByte, &energy)
	if errJson != nil{
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	//fmt.Fprintf(w, "test1: %q", energyByte)
	for _, value := range energy.Payload.Days{
		fmt.Fprintf(w, "Here")
		_, ok := dayArray[value.Day]
		if ok == true{
			continue
		}
		day, err1 := json.Marshal(value)
		if err1 != nil{
			fmt.Println("error:", err1)
		}
		fmt.Fprintf(w, "%v", value.Cycles[0])
		c := appengine.NewContext(r)
		g := models.DayDataStore{
			Email: "jamier.net@gmail.com",
			Day: day,
			Date: value.Day,
			/*Total_heating_time: value.Total_heating_time,
			Total_cooling_time: value.Total_cooling_time,
			//Total_fan_cooling_time: value.Total_fan_cooling_time,
			//Total_humidifier_time: value.Total_humidifier_time,
			//	Total_dehumidifier_time: value.Total_dehumidifier_time,
			Leafs: value.Leafs,*/
		}
		
		_, err := datastore.Put(c, datastore.NewIncompleteKey(c, "DayDataStore", nil), &g)
	    if err != nil {
	        http.Error(w, err.Error(), http.StatusInternalServerError)
	        return
	    }
	}
}

func export(w http.ResponseWriter, r *http.Request){
	cookie, err := r.Cookie("email")
	if err != nil{
		http.Redirect(w, r, "/login", 301)
		return
	}
    // Gets the last 30 days
    energy := nestdatastore.GetValuesFromDataStore(30, cookie.Value, w, r)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set(`Content-Disposition`, `attachment; filename="export.csv"`);
	out := csv.NewWriter(w)
	out.Write([]string{"Date", "Leafs", "Recent_avg_used", "Total_cooling_time", 
		"Total_dehumidifier_time", "Total_fan_cooling_time", "Total_heating_time",
		"Total_humidifier_time", "Usage_over_avg", "Whodunit"})
	for _, value := range energy{
    	out.Write([]string {string(value.Date), strconv.Itoa(value.Leafs), 
    		strconv.Itoa(value.Recent_avg_used), strconv.Itoa(value.Total_cooling_time),
    		strconv.Itoa(value.Total_dehumidifier_time), strconv.Itoa(value.Total_fan_cooling_time),
    		strconv.Itoa(value.Total_heating_time), strconv.Itoa(value.Total_humidifier_time),
    		strconv.Itoa(value.Usage_over_avg), nestdatastore.Whodunit(value.Whodunit)})
    	//fmt.Fprintf(w, "%s", value)
    }
	out.Flush()
}

func init(){
	http.HandleFunc("/", root)
	http.HandleFunc("/sign", sign)
	http.HandleFunc("/login", login)
	http.HandleFunc("/loginSubmit", loginSubmit)
	http.HandleFunc("/api/energy", api_energy)
	http.HandleFunc("/export", export)

	/*userkey := models.Key{
		Key: "test",
		Version: 123,
		Timestamp: 456,
	}

	userkeys := []models.Key{userkey}
	userkeys = append(userkeys, userkey)
	userkeys = append(userkeys, userkey)

	testKey := models.Keys{userkeys}
	b, err := json.Marshal(userkeys)
	if err != nil{
		fmt.Println("error: ", err)
	}
	fmt.Printf("%s", b)
	fmt.Println()
	q, err := json.Marshal(testKey)
	fmt.Printf("%s", q)
	fmt.Println()

	str := "device.02AA01AB44120G73"

	strSplit := strings.Split(str, ".")[1]

	fmt.Printf("%s", strSplit)
	fmt.Println()
	/* This is doing the initial call */
	/*
	resp, err := http.PostForm("https://home.nest.com/user/login", url.Values{"username":{"jamier.net@gmail.com"}, "password": {"21phabet"}})

	if err != nil{
		fmt.Printf("Error!!")
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("Initial Call ---->")
	fmt.Printf("%s", body)
	fmt.Println("---->")
	
	var nestLogin NestLogin
	errJson := json.Unmarshal(body, &nestLogin)
	if errJson != nil{
		log.Fatal(errJson)
	}

	//client := &http.Client{}
	//s := `{"keys":[{"key":"user.188262","version":-843004088,"timestamp":1356287364000},{"key":"user_settings.188262","version":1496039529,"timestamp":1356287392615},{"key":"user_alert_dialog.188262","version":-1320746469,"timestamp":1356287465000},{"key":"structure.ad4daf00-4d2e-11e2-a4a7-12313d19168a","version":-1612033796,"timestamp":1357218210000}]}`
	//s := `{"keys":[{"key":"user.188262","version":-843004088,"timestamp":1356287364000},{"key":"user_settings.188262","version":1496039529,"timestamp":1356287392615},{"key":"user_alert_dialog.188262","version":-1320746469,"timestamp":1356287465000},{"key":"structure.ad4daf00-4d2e-11e2-a4a7-12313d19168a","version":-1612033796,"timestamp":1357218210000},{"key":"device.02AA01AB44120G73","version":-2074707235,"timestamp":1357218210000},{"key":"shared.02AA01AB44120G73","version":543477440,"timestamp":1357218271000},{"key":"message.02AA01AB44120G73","version":826888083,"timestamp":1356287487000},{"key":"schedule.02AA01AB44120G73","version":-1505049159,"timestamp":1357163807000},{"key":"track.02AA01AB44120G73","version":-568959740,"timestamp":1357218469569},{"key":"energy_latest.02AA01AB44120G73","version":1,"timestamp":1357118949157}]}`
	//s := `{"keys":[{"key":"user.188262","version":-843004088,"timestamp":1356287364000},{"key":"user_alert_dialog.188262","version":-1320746469,"timestamp":1356287465000},{"key":"user_settings.188262","version":1496039529,"timestamp":1356287392615},{"key":"structure.ad4daf00-4d2e-11e2-a4a7-12313d19168a","version":1800356857,"timestamp":1360374384000},{"key":"device.02AA01AB44120G73","version":1348169463,"timestamp":1360446485000},{"key":"shared.02AA01AB44120G73","version":1309307191,"timestamp":1360454917000},{"key":"schedule.02AA01AB44120G73","version":-79023839,"timestamp":1360387822000},{"key":"track.02AA01AB44120G73","version":583994169,"timestamp":1360458455726},{"key":"energy_latest.02AA01AB44120G73"}]}`
	/*s := `{"keys":[{"key":"user.188262","version":-843004088,"timestamp":1356287364000},{"key":"user_alert_dialog.188262","version":-1320746469,"timestamp":1356287465000},{"key":"user_settings.188262","version":1496039529,"timestamp":1356287392615},{"key":"device.02AA01AB44120G73","version":1348169463,"timestamp":1360446485000},{"key":"shared.02AA01AB44120G73","version":1309307191,"timestamp":1360454917000},{"key":"schedule.02AA01AB44120G73","version":-79023839,"timestamp":1360387822000},{"key":"track.02AA01AB44120G73","version":583994169,"timestamp":1360458455726},{"key":"energy_latest.02AA01AB44120G73"}]}`
	req, err := http.NewRequest("POST", nestLogin.Urls.Transport_url + "/v2/subscribe", strings.NewReader(s))
	req.ContentLength = int64(len(s)) 
	req.Header.Add("Authorization", "Basic " + nestLogin.Access_token)
	req.Header.Add("X-nl-protocol-version", "1")
	req.Header.Add("X-nl-client-timestamp", "1360458454279")
	req.Header.Add("X-nl-session-id", "136045845235226189601.188525558")
	req.Header.Add("X-nl-subscribe-timeout", "60")
	req.Header.Add("User-Agent", "Nest/3.0.15 (iOS) os=6.0 platform=iPad3,1")
	resp1, err := client.Do(req)

        if err != nil{
                fmt.Printf("Error!!")
                log.Fatal(err)
        }

        defer resp1.Body.Close()
        body1, err := ioutil.ReadAll(resp1.Body)
	fmt.Println("POST ---->")
	fmt.Println("POST ---->")
        fmt.Printf("%s", body1)
        fmt.Println("POST ---->")
        fmt.Println("POST ---->")
        fmt.Println(resp)
        fmt.Println("POST resp ---->")
        fmt.Println("POST resp---->")
/*	
	s1 := `{"keys":[{"key":"user.188262","version":-843004088,"timestamp":1356287364000},{"key":"user_alert_dialog.188262","version":-1320746469,"timestamp":1356287465000},{"key":"user_settings.188262","version":1496039529,"timestamp":1356287392615},{"key":"structure.ad4daf00-4d2e-11e2-a4a7-12313d19168a","version":1800356857,"timestamp":1360374384000},{"key":"device.02AA01AB44120G73","version":1348169463,"timestamp":1360446485000},{"key":"shared.02AA01AB44120G73","version":1309307191,"timestamp":1360454917000},{"key":"schedule.02AA01AB44120G73","version":-79023839,"timestamp":1360387822000},{"key":"track.02AA01AB44120G73","version":583994169,"timestamp":1360458455726},{"key":"energy_latest.02AA01AB44120G73"}]}`
	req1, err := http.NewRequest("POST", nestLogin.Urls.Transport_url + "/v2/subscribe", strings.NewReader(s1))
	req1.ContentLength = int64(len(s1)) 
	req1.Header.Add("Authorization", "Basic " + nestLogin.Access_token)
	req1.Header.Add("X-nl-subscribe-timeout", "8")
	req1.Header.Add("X-nl-protocol-version", "1")
	req1.Header.Add("X-nl-client-timestamp", "1360458454279")
	req1.Header.Add("X-nl-session-id", "136045845235226189601.188525558")

	req1.Header.Add("User-Agent", "Nest/3.0.15 (iOS) os=6.0 platform=iPad3,1")
	resp2, err := client.Do(req1)

        if err != nil{
                fmt.Printf("Error!!")
                log.Fatal(err)
        }

        defer resp2.Body.Close()
        body2, err := ioutil.ReadAll(resp2.Body)
	fmt.Println("POST ---->")
	fmt.Println("POST ---->")
        fmt.Printf("%s", body2)
        fmt.Println("POST ---->")
        fmt.Println("POST ---->")
        fmt.Println(resp)
        fmt.Println("POST resp ---->")
        fmt.Println("POST resp---->")
        {{define "body"}}
{{range .}}
      <h1>{{.Day}}</h1>
      {{range .Events}}
      	<pre>{{.Heat_Temp}} -- {{ctof .Heat_Temp}}</pre>
      {{end}}
{{end}}
{{end}}
*/	
}
