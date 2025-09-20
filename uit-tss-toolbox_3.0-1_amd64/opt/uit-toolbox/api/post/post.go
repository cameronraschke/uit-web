package post

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"net/http"

	// "net/http/httputil"
	"strconv"
	"time"

	"uit-toolbox/api/database"

	"github.com/google/uuid"
)

type RemoteTable struct {
	Tagnumber         *int       `sql:"tagnumber"`
	JobQueued         *string    `sql:"job_queued"`
	JobQueuedPosition *int       `sql:"job_queued_position"`
	JobActive         *bool      `sql:"job_queued_position"`
	CloneMode         *string    `sql:"clone_mode"`
	EraseMode         *string    `sql:"erase_mode"`
	LastJobTime       *time.Time `sql:"last_job_time"`
	Present           *time.Time `sql:"present"`
	PresentBool       *bool      `sql:"present_bool"`
	Status            *string    `sql:"status"`
	KernelUpdated     *bool      `sql:"kernel_updated"`
	BatteryCharge     *int       `sql:"battery_charge"`
	BatteryStatus     *string    `sql:"battery_status"`
	Uptime            *int       `sql:"uptime"`
	CpuTemp           *int       `sql:"cpu_temp"`
	DiskTemp          *int       `sql:"disk_temp"`
	MaxDiskTemp       *int       `sql:"max_disk_temp"`
	WattsNow          *int       `sql:"watts_now"`
	NetworkSpeed      *int       `sql:"network_speed"`
}

type FormJobQueue struct {
	Tagnumber string `json:"job_queued_tagnumber"`
	JobQueued string `json:"job_queued_select"`
}

func UpdateRemoteJobQueued(ctx context.Context, req *http.Request, db *sql.DB, key string) error {
	// Parse request body JSON
	var j FormJobQueue
	err := json.NewDecoder(req.Body).Decode(&j)
	if err != nil {
		return errors.New("Cannot parse request body JSON: " + err.Error())
	}
	defer req.Body.Close()

	tag := j.Tagnumber
	var tagnumber int
	if len(tag) > 0 {
		tagnumber, err = strconv.Atoi(tag)
		if err != nil {
			return errors.New("Tagnumber cannot be converted to integer: " + j.Tagnumber)
		}
	}
	value := j.JobQueued

	log.Println("Updating job_queued for tagnumber " + j.Tagnumber + " to value " + j.JobQueued)

	// Commit to DB
	if key == "job_queued" {
		err := database.UpdateDB(ctx, db, "UPDATE remote SET job_queued = $1 WHERE tagnumber = $2", value, tagnumber)
		if err != nil {
			return errors.New("Database error: " + err.Error())
		}
		return nil
	}

	return errors.New("Unknown key: " + key)
}

type FormClientImages struct {
	Tagnumber *string `json:"tagnumber"`
	ImageNote *string `json:"note"`
}

func UpdateClientImages(ctx context.Context, req *http.Request, db *sql.DB, key string) error {

	// dump, err := httputil.DumpRequestOut(req, true)
	// if err != nil {
	// 	return errors.New("Cannot dump HTTP request: " + err.Error())
	// }

	// fmt.Println("--- Dumped HTTP Request ---")
	// fmt.Println(string(dump))
	// fmt.Println("---------------------------")

	// Parse request body
	err := req.ParseMultipartForm(64 << 20)
	if err != nil {
		return errors.New("Cannot parse form: " + err.Error())
	}
	files := req.MultipartForm.File["userfile"]
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return errors.New("cannot open uploaded file for reading")
		}
		defer file.Close()

		uploadedImage, imageType, err := image.Decode(file)
		if err != nil {
			return errors.New("cannot decode uploaded file")
		}

		var b bytes.Buffer
		err = jpeg.Encode(&b, uploadedImage, &jpeg.Options{Quality: 100})
		if err != nil {
			return errors.New("cannot encode uploaded file")
		}

		byteSlice := b.Bytes()
		EncodedImageData := base64.StdEncoding.EncodeToString([]byte(byteSlice))

		note := req.FormValue("note")

		uuidBytes := uuid.New()
		uuid := uuidBytes.String()

		time := time.Now().Format("2025-08-22 00:00:00.000")

		fmt.Println(EncodedImageData)
		fmt.Println(imageType)
		fmt.Println(uuid)
		fmt.Println(time)
		fmt.Println(note)

		// var j FormClientImages
		// err := json.NewDecoder(req.Body).Decode(&j)
		// if err != nil {
		//   return errors.New("Cannot parse request body JSON: " + err.Error())
		// }
		defer req.Body.Close()

		// tagnumber := j.Tagnumber
		// value := j.JobQueued

		// // Commit to DB
		// if (key == "job_queued") {
		//   err := database.UpdateDB(ctx, db, "UPDATE remote SET job_queued = $1 WHERE tagnumber = $2", value, tagnumber)
		//   if err != nil {
		//     return errors.New("Database error: " + err.Error())
		//   }
		//   return nil
		// }
		return nil
	}

	return errors.New("No files in upload: " + key)
}
