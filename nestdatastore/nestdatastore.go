package nestdatastore

import(
	"appengine"
	"appengine/datastore"
	"models"
	"encoding/json"
	"net/http"
)

func GetDayArray(energy []models.DayDataStore) map[string]models.Day{
	var dayArray map[string] models.Day
	dayArray = make(map[string]models.Day)
	for _, value := range energy{
    	var day models.Day
    	json.Unmarshal(value.Day, &day)
    	dayArray[day.Day] = day
    }
    return dayArray
}

func GetValuesFromDataStoreForImport(email string, w http.ResponseWriter, r *http.Request) []models.DayDataStore{
    c := appengine.NewContext(r)
    q := datastore.NewQuery("DayDataStore").
        Filter("Email =", email).
        Order("-Date")
    energy := make([]models.DayDataStore, 0, 10)
    if _, err := q.GetAll(c, &energy); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return nil
    }
    return energy
}

func GetValuesFromDataStore(count int, email string, w http.ResponseWriter, r *http.Request) []models.DayDataStore{
	c := appengine.NewContext(r)
    q := datastore.NewQuery("DayDataStore").
        Filter("Email =", email).
        Order("Date").
        Limit(count)

    energy := make([]models.DayDataStore, 0, count)
    if _, err := q.GetAll(c, &energy); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return nil
    }
    return energy
}

func Whodunit(whodunit int) string{
    // whodunit 2 = Away caused by energy usage below the weekly average
    // whodunit 3 = Auto-Away caused energy usage below the weekly average
    // whodunit 0 = Your adjustment caused energy usage above the weekly average
    // whodunit 1 = This day's weather caused energy usage above/below the weekly average
    switch whodunit{
        case 0:
            return "Your adjustment caused energy usage above the weekly average"
        case 1:
            return "This day's weather caused energy usage above/below the weekly average"
        case 2:
            return "Away caused by energy usage below the weekly average"
        case 3:
            return "Auto-Away caused energy usage below the weekly average"
    }
    return ``

}