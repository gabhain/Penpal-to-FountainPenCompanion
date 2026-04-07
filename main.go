package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type CollectedInk struct {
	BrandName    string  `json:"brand_name"`
	InkName      string  `json:"ink_name"`
	Kind         string  `json:"kind"` // bottle, sample, cartridge
	Volume       float64 `json:"volume,omitempty"`
	Swabbed      bool    `json:"swabbed"`
	Comment      string  `json:"comment,omitempty"`
	PurchaseDate string  `json:"purchase_date,omitempty"`
}

type CollectedPen struct {
	Brand         string  `json:"brand"`
	Model         string  `json:"model"`
	Nib           string  `json:"nib"`
	Color         string  `json:"color"`
	Material      string  `json:"material,omitempty"`
	FillingSystem string  `json:"filling_system,omitempty"`
	Price         float64 `json:"price,omitempty"`
	Comment       string  `json:"comment,omitempty"`
}

type InkRequest struct {
	Data struct {
		Type       string       `json:"type"`
		Attributes CollectedInk `json:"attributes"`
	} `json:"data"`
}

type PenRequest struct {
	Data struct {
		Type       string       `json:"type"`
		Attributes CollectedPen `json:"attributes"`
	} `json:"data"`
}

var (
	logBinding = binding.NewString()
	logOutput  *widget.Entry
)

func main() {
	myApp := app.NewWithID("com.gabhain.penpal-importer")
	myWindow := myApp.NewWindow("PenPal to FPC Importer")

	tokenEntry := widget.NewEntry()
	tokenEntry.SetPlaceHolder("e.g., 123.abcde...")

	selectedFiles := []fyne.URI{}
	filesLabel := widget.NewLabel("No files selected")

	logOutput = widget.NewMultiLineEntry()
	logOutput.Bind(logBinding)

	selectBtn := widget.NewButton("Select CSV Files", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			if reader == nil {
				return
			}
			selectedFiles = append(selectedFiles, reader.URI())
			filesLabel.SetText(fmt.Sprintf("%d file(s) selected", len(selectedFiles)))
			log(fmt.Sprintf("Selected: %s", reader.URI().Name()))
		}, myWindow)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
		fd.Show()
	})

	uploadBtn := widget.NewButton("Upload to FPC", func() {
		token := tokenEntry.Text
		if token == "" {
			dialog.ShowInformation("Error", "Please enter an API Token", myWindow)
			return
		}
		if len(selectedFiles) == 0 {
			dialog.ShowInformation("Error", "Please select at least one CSV file", myWindow)
			return
		}

		go func() {
			for _, fileURI := range selectedFiles {
				processFile(fileURI, token)
			}
			log("All finished!")
		}()
	})

	content := container.NewVBox(
		widget.NewLabel("FPC API Token (id.token):"),
		tokenEntry,
		selectBtn,
		filesLabel,
		uploadBtn,
		widget.NewLabel("Logs:"),
		container.NewStack(logOutput),
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(600, 400))
	myWindow.ShowAndRun()
}

func log(msg string) {
	timestamp := time.Now().Format("15:04:05")
	current, _ := logBinding.Get()
	logBinding.Set(current + fmt.Sprintf("[%s] %s\n", timestamp, msg))
}

func processFile(uri fyne.URI, token string) {
	log(fmt.Sprintf("Processing %s...", uri.Name()))

	f, err := storage.OpenFileFromURI(uri)
	if err != nil {
		log(fmt.Sprintf("Error opening file: %v", err))
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	headers, err := reader.Read()
	if err != nil {
		log(fmt.Sprintf("Error reading headers: %v", err))
		return
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[h] = i
	}

	if _, ok := headerMap["attributes_list_option_ink_attributes"]; ok {
		log("Detected INK collection")
		processInks(reader, headerMap, token)
	} else if _, ok := headerMap["bodymaterial_option_pen_material"]; ok {
		log("Detected PEN collection")
		processPens(reader, headerMap, token)
	} else if _, ok := headerMap["Company"]; ok {
		log("Detected PEN collection (Custom headers)")
		processPens(reader, headerMap, token)
	} else {
		log("Could not detect collection type (unknown headers)")
	}
}

func processInks(reader *csv.Reader, headerMap map[string]int, token string) {
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log(fmt.Sprintf("Error reading record: %v", err))
			continue
		}

		ink := CollectedInk{
			BrandName: getVal(record, headerMap, "brand_text"),
			InkName:   getVal(record, headerMap, "color_name_text"),
			Kind:      mapKind(getVal(record, headerMap, "vessel_option_ink_vessel")),
			Comment:   getVal(record, headerMap, "notes_text"),
		}

		if ink.BrandName == "" || ink.InkName == "" {
			continue
		}

		uploadInk(ink, token)
	}
}

func mapKind(v string) string {
	v = strings.ToLower(v)
	if strings.Contains(v, "sample") {
		return "sample"
	}
	if strings.Contains(v, "cartridge") {
		return "cartridge"
	}
	return "bottle" // default
}

func uploadInk(ink CollectedInk, token string) {
	url := "https://www.fountainpencompanion.com/api/v1/collected_inks.json"
	
	reqBody := InkRequest{}
	reqBody.Data.Type = "collected_ink"
	reqBody.Data.Attributes = ink
	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if !strings.HasPrefix(token, "Bearer ") {
		token = "Bearer " + token
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log(fmt.Sprintf("Failed to upload ink %s: %v", ink.InkName, err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		log(fmt.Sprintf("Successfully uploaded ink: %s %s", ink.BrandName, ink.InkName))
	} else {
		body, _ := io.ReadAll(resp.Body)
		log(fmt.Sprintf("Failed to upload ink %s: %s (Status: %d)", ink.InkName, string(body), resp.StatusCode))
	}
}

func processPens(reader *csv.Reader, headerMap map[string]int, token string) {
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log(fmt.Sprintf("Error reading record: %v", err))
			continue
		}

		pen := CollectedPen{
			Brand:         getVal(record, headerMap, "brand_text", "Company"),
			Model:         getVal(record, headerMap, "style_text", "Model"),
			Nib:           getVal(record, headerMap, "nib_size_display_text", "Nib"),
			Color:         getVal(record, headerMap, "color_text", "Color"),
			Material:      getVal(record, headerMap, "bodymaterial_option_pen_material", "Material"),
			FillingSystem: getVal(record, headerMap, "fill_option_fill_system", "Filling", "Filling System"),
		}

		if pen.Brand == "" || pen.Model == "" {
			continue
		}

		uploadPen(pen, token)
	}
}

func uploadPen(pen CollectedPen, token string) {
	url := "https://www.fountainpencompanion.com/api/v1/collected_pens"
	
	reqBody := PenRequest{}
	reqBody.Data.Type = "collected_pen"
	reqBody.Data.Attributes = pen
	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if !strings.HasPrefix(token, "Bearer ") {
		token = "Bearer " + token
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log(fmt.Sprintf("Failed to upload pen %s: %v", pen.Model, err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		log(fmt.Sprintf("Successfully uploaded pen: %s %s", pen.Brand, pen.Model))
	} else if resp.StatusCode == http.StatusNotFound {
		log(fmt.Sprintf("Failed to upload pen %s: FPC currently has a read-only API for pens.", pen.Model))
		log("To import pens, please use the web interface:")
		log("https://www.fountainpencompanion.com/collected_pens/import")
	} else {
		body, _ := io.ReadAll(resp.Body)
		log(fmt.Sprintf("Failed to upload pen %s: %s (Status: %d)", pen.Model, string(body), resp.StatusCode))
	}
}

func getVal(record []string, headerMap map[string]int, cols ...string) string {
	for _, col := range cols {
		idx, ok := headerMap[col]
		if ok && idx < len(record) && record[idx] != "" {
			return record[idx]
		}
	}
	return ""
}
