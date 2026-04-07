# PenPal to Fountain Pen Companion Importer

This Go application allows you to import your ink and pen collections from [penpal.network](https://penpal.network) to [fountainpencompanion.com](https://www.fountainpencompanion.com) using their API.

## Features

- **Automatic Detection**: Automatically detects whether a CSV file contains inks or pens based on headers.
- **Bulk Upload**: Uploads each item to your FPC account.
- **Simple GUI**: Easy-to-use interface to select files and enter your API token.

## Prerequisites

- **Go**: You need to have Go installed to build the application.
- **FPC API Token**: You can find or create your API token on the [Fountain Pen Companion account page](https://www.fountainpencompanion.com/authentication_tokens).
  - The token should be in the format `id.token` (e.g., `123.abcde...`).

## Usage

1. **Install Dependencies**:
   ```bash
   go mod tidy
   ```

2. **Build and Run**:
   ```bash
   go run main.go
   ```

3. **Follow the GUI**:
   - Enter your **API Token**.
   - Click **Select CSV Files** and choose your PenPal exports.
   - Click **Upload to FPC**.

## Important Notes on Pen Uploads

Based on the current public Fountain Pen Companion API documentation/code:
- **Inks**: The API fully supports creating new ink entries (`POST /api/v1/collected_inks`).
- **Pens**: The public `v1` API currently only supports *reading* pens (`GET /api/v1/collected_pens`). This tool will *attempt* to upload to that endpoint, but it may fail if the site's API is strictly read-only for pens. If it fails, you may need to use the web-based import feature on the FPC site or contact the site creator for a bulk import.

## Mapping

The tool maps the following fields from PenPal:

### Inks
- `brand_text` -> Brand
- `color_name_text` -> Ink Name
- `vessel_option_ink_vessel` -> Type (Bottle, Sample, or Cartridge)
- `notes_text` -> Comment

### Pens
- `brand_text` -> Brand
- `style_text` -> Model
- `nib_size_display_text` -> Nib
- `color_text` -> Color
- `bodymaterial_option_pen_material` -> Material
- `fill_option_fill_system` -> Filling System

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
