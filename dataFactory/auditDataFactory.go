package dataFactory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	pgx "github.com/jackc/pgx/v4"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/klog"
)

func Insert(items []audit.Event) {
	defer func() {
		if v := recover(); v != nil {
			klog.Errorln("capture a panic:", v)
		}
	}()

	queryInsertTimeseriesData := `
	INSERT INTO audit (ID, USERNAME, USERAGENT , NAMESPACE , APIGROUP , APIVERSION , RESOURCE , NAME , STAGE , STAGETIMESTAMP , VERB, CODE , STATUS , REASON , MESSAGE ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);
	`

	/********************************************/
	/* Batch Insert into TimescaelDB            */
	/********************************************/
	//create batch
	batch := &pgx.Batch{}
	numInserts := len(items)
	//load insert statements into batch queue

	for _, event := range items {
		if event.ObjectRef.Resource == "events" {
			if event.RequestObject == nil {
				klog.Errorf("No request body for event")
			} else if strings.Contains(string(event.RequestObject.Raw), "\"hypercloud\":\"claim\"") {
				var body map[string]interface{}
				if err := json.Unmarshal([]byte(string(event.RequestObject.Raw)), &body); err != nil {
					fmt.Errorf(err.Error())
					continue
				}
				email := valueFromKey(body, "reportingController")
				email = strings.Replace(email, "-", "@", -1)
				batch.Queue(queryInsertTimeseriesData, event.AuditID,
					email,
					event.UserAgent,
					NewNullString(valueFromKey(body, "metadata", "namespace")),
					NewNullString("claim.tmax.io"),
					NewNullString("v1alpha1"),
					parseClaim(valueFromKey(body, "metadata", "name")),
					valueFromKey(body, "reportingInstance"),
					event.Stage,
					event.StageTimestamp.Time,
					valueFromKey(body, "action"),
					event.ResponseStatus.Code,
					event.ResponseStatus.Status,
					valueFromKey(body, "reason"),
					event.ResponseStatus.Message)
			}
			continue
		}

		batch.Queue(queryInsertTimeseriesData, event.AuditID,
			event.User.Username,
			event.UserAgent,
			NewNullString(event.ObjectRef.Namespace),
			NewNullString(event.ObjectRef.APIGroup),
			NewNullString(event.ObjectRef.APIVersion),
			event.ObjectRef.Resource,
			event.ObjectRef.Name,
			event.Stage,
			event.StageTimestamp.Time,
			event.Verb,
			event.ResponseStatus.Code,
			event.ResponseStatus.Status,
			event.ResponseStatus.Reason,
			event.ResponseStatus.Message)
	}
	//send batch to connection pool
	br := Dbpool.SendBatch(context.TODO(), batch)
	//execute statements in batch queue
	for i := 0; i < numInserts; i++ {
		_, err := br.Exec()
		if err != nil {
			klog.Errorln(err)
			// os.Exit(1)
		}
	}
	// klog.Infoln("Successfully batch inserted data n")

	//Compare length of results slice to size of table
	klog.Infof("size of results: %d\n", numInserts)
	//check size of table for number of rows inserted
	// result of last SELECT statement
	var rowsInserted int
	err := br.QueryRow().Scan(&rowsInserted)
	klog.Infof("size of table: %d\n", rowsInserted)

	err = br.Close()
	if err != nil {
		klog.Errorf("Unable to closer batch %v\n", err)
	}
}

func Get(query string) (audit.EventList, int64) {
	defer func() {
		if v := recover(); v != nil {
			klog.Errorln("capture a panic:", v)
		}
	}()

	// role check
	rows, err := Dbpool.Query(context.TODO(), query)
	if err != nil {
		klog.Errorf("Unable to execute query %v\n", err)
	}
	defer rows.Close()

	var row_count int64
	// Print rows returned and fill up results slice for later use
	var eventList audit.EventList
	for rows.Next() {
		var temp_namespace, temp_apigroup, temp_apiversion sql.NullString
		event := audit.Event{
			ObjectRef:      &audit.ObjectReference{},
			ResponseStatus: &metav1.Status{},
		}
		err = rows.Scan(
			&event.AuditID,
			&event.User.Username,
			&event.UserAgent,
			&temp_namespace,  //&event.ObjectRef.Namespace,
			&temp_apigroup,   //&event.ObjectRef.APIGroup,
			&temp_apiversion, //&event.ObjectRef.APIVersion,
			&event.ObjectRef.Resource,
			&event.ObjectRef.Name,
			&event.Stage,
			&event.StageTimestamp.Time,
			&event.Verb,
			&event.ResponseStatus.Code,
			&event.ResponseStatus.Status,
			&event.ResponseStatus.Reason,
			&event.ResponseStatus.Message,
			&row_count)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to scan %v\n", err)
			os.Exit(1)
		}
		if temp_namespace.Valid {
			event.ObjectRef.Namespace = temp_namespace.String
		} else {
			event.ObjectRef.Namespace = ""
		}

		if temp_apigroup.Valid {
			event.ObjectRef.APIGroup = temp_apigroup.String
		} else {
			event.ObjectRef.APIGroup = ""
		}

		if temp_apiversion.Valid {
			event.ObjectRef.APIVersion = temp_apiversion.String
		} else {
			event.ObjectRef.APIVersion = ""
		}
		eventList.Items = append(eventList.Items, event)
	}
	// Any errors encountered by rows.Next or rows.Scan will be returned here
	if rows.Err() != nil {
		klog.Errorf("rows Error: %v\n", rows.Err())
	}
	// use results here…
	eventList.Kind = "EventList"
	eventList.APIVersion = "audit.k8s.io/v1"

	return eventList, row_count
}

func GetMemberList(query string) ([]string, int64) {
	defer func() {
		if v := recover(); v != nil {
			klog.Errorln("capture a panic:", v)
		}
	}()

	rows, err := Dbpool.Query(context.TODO(), query)
	if err != nil {
		klog.Error(err)
	}
	defer rows.Close()

	memberList := []string{}
	var row_count int64

	for rows.Next() {
		var member string
		err := rows.Scan(
			&member,
			&row_count)
		if err != nil {
			rows.Close()
			klog.Error(err)
		}
		memberList = append(memberList, member)
	}

	return memberList, row_count
}

func NewNullString(s string) sql.NullString {
	if s == "null" {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func valueFromKey(r map[string]interface{}, key ...string) string {
	if len(key) == 1 {
		return r[key[0]].(string)
	} else if len(key) == 2 {
		return r[key[0]].(map[string]interface{})[key[1]].(string)
	} else {
		fmt.Errorf("Not Support for 3 or more parameters.")
	}
	return ""
}

func parseClaim(str string) string {
	s := strings.Split(str, "-")
	return s[0]
}
