# go-bedrock-nbt-api
AWS Lambda function for converting `.mcstructure` to Java NBT via an HTTP API payload, designed specifically for seamless frontend client integrations.

## Overview
A serverless Go function designed to be hooked to AWS API Gateway. It receives raw binary `.mcstructure` data as `application/octet-stream` via `POST`, decodes it in memory, performs the structural conversion through the embedded `go-bedrock-nbt-converter`, and responds with a binary `converted.nbt` Java Edition Structure block format.

## Setup Instructions (AWS SAM CLI)

This project is configured natively for AWS SAM (Serverless Application Model). The included `template.yml` and `openapi.yml` will automatically set up the API Gateway and correctly configure the required **Binary Media Types** routing to avoid file corruption.

### 1. Build the Project
We use a simple PowerShell script to cross-compile the Go application into the standard AWS Linux AMD64 format (`bootstrap`) and compress it into a `lambda.zip` file.
```powershell
# In PowerShell:
.\build.ps1
```

### 2. Deploy to AWS
Now that the `lambda.zip` is ready, use AWS SAM to automatically provision the API Gateway and upload the Lambda function. Run the following command and follow the prompts:
```bash
sam deploy --guided --resolve-s3
```
Follow the interactive prompts (Stack Name, AWS Region, allow SAM to create IAM roles & API Gateway resources).
SAM will deploy the CloudFormation stack and at the end print out an **Outputs** table containing your `ApiEndpoint` URL.

---

## Simple Client Example (Browser JavaScript)
Because this API directly parses and replies with browser File/Blob binary buffers, the client frontend code is incredibly simple:

```javascript
const fileInput = document.getElementById('fileInput');

async function convertFile() {
    const file = fileInput.files[0];
    if (!file) return;

    // The API automatically determines the format from the filename extension:
    // .mcstructure (Bedrock Structure)
    // .mcworld (Bedrock World Zip)
    // .schem (Java Sponge Schematic / WorldEdit)
    // .litematic (Java Litematica)
    
    // Pass the original filename to the API:
    let apiUrl = `https://<your-api-url-here>/?filename=${encodeURIComponent(file.name)}`;
    
    // Optionally change the downloaded output name:
    // apiUrl += `&output=my_awesome_converted_build.nbt`;

    // .mcworld bounds extraction is still fully supported:
    // apiUrl += `&min_x=-50&max_x=50&min_y=-64&max_y=320&min_z=-50&max_z=50&dimension=0`;

    const response = await fetch(apiUrl, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/octet-stream' 
        },
        body: file // Direct binary upload
    });

    if (response.ok) {
        // The API instantly returns a file download buffer 
        const nbtBlob = await response.blob();
        
        // Use the filename provided by the API's Content-Disposition header if possible, or fallback
        let downloadName = 'converted.nbt';
        const disposition = response.headers.get('Content-Disposition');
        if (disposition && disposition.indexOf('filename=') !== -1) {
            const matches = /filename="([^"]*)"/.exec(disposition);
            if (matches != null && matches[1]) downloadName = matches[1];
        }

        // Trigger generic file download script in Browser
        const url = URL.createObjectURL(nbtBlob);
        const a = document.createElement('a');
        a.href = url;
        a.download = downloadName;
        document.body.appendChild(a);
        a.click();
        URL.revokeObjectURL(url);
    } else {
        console.error("Conversion failed:", await response.text());
    }
}
```
