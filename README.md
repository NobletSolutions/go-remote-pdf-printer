# remote-pdf-printer
Converts a URL or HTML into PDF via Headless Google Chrome instance

# End Points

PDF
* /pdf [POST]
* /pdf/:file [GET]
* /preview [POST]
* /preview/:file [GET]


Both /pdf and /preview POST expect `application/json` data.

```
{
    "data": [...], // array of strings, submit HTML this way though nothing stops you from submitting an external URL
    "url": [...], // array of urls
    "download": boolean, // default false - return the file directly if true
    "header": string, // header content - if set marginTop is required
    "footer": string, // footer content - if set marginBottom is required
    "marginTop": float,
    "marginBottom": float,
    "marginLeft": float,
    "marginRight": float,
    "paperSize": [float,float] // [width,height]
}
```

# /pdf

When download is set to false the return value is

```
{
    "components": [
        "http://localhost:8080/pdfs/0-793062911.pdf",
        "http://localhost:8080/pdfs/1-1442655579.pdf"
    ],
    "pdf": "2844005942-combined.pdf",
    "url": "http://localhost:8080/pdfs/2844005942-combined.pdf"
}
```

# /preview

The return value
```
{
    "basename": "2378371505-combined",
    "images": [
        "http://localhost:8080/preview/2378371505-combined-1.jpg",
        "http://localhost:8080/preview/2378371505-combined-2.jpg"
    ],
    "pages": 2,
    "pdfInfo": {
        "custom_metadata": "no",
        "encrypted": "no",
        "file_size": "177025 bytes",
        "form": "none",
        "javascript": "no",
        "metadata_stream": "no",
        "optimized": "no",
        "page_rot": "0",
        "page_size": "612 x 792 pts (letter)",
        "pages": "2",
        "pdf_version": "1.4",
        "suspects": "no",
        "tagged": "no",
        "userproperties": "no"
    },
    "success": true
}
```

# Service Configuration

There are a number of environment variables that can be set to control the service

| Environment Variable                   | Default Value                      |
| -------------------------------------- | ---------------------------------- |
| REMOTE_PDF_ROOT_DIRECTORY              | The process CWD                    |
| REMOTE_PDF_DEBUG_HEADER_STYLE_TEMPLATE | css/default-header.css.txt |
| REMOTE_PDF_PORT                        | 3000                               |
| REMOTE_PDF_LISTEN                      | 127.0.0.1                          |
| REMOTE_PDF_CHROME_URI                  | 127.0.0.1:1337                     |
| REMOTE_PDF_USE_TLS                     | false                              |
| REMOTE_PDF_TLS_CERT_PATH               | nil - required if TLS is true      |
| REMOTE_PDF_TLS_KEY_PATH                | nil - required if TLS is true      |
| REMOTE_PDF_LOG_PATH                    | /var/log - currently unused        |


# Docker Container

A Dockerfile is provided that builds a container to use.

Running `podman build . -t localhost/remote-pdf-printer:latest` will build and tag the container

It is ideal to use a storage volume for the files

To run it with a local chrome instance

You'll need a headless chrome instance running and listening on port 1337

`podman run -name remote-pdf-printer -e REMOTE_PDF_LISTEN=0.0.0.0 -e REMOTE_PDF_CHROME_URI="host.containers.internal:1337" -e REMOTE_PDF_USE_TLS=false -p 8080:3000 -v ./files:/app/files:z localhost/remote-pdf-printer:latest`

Will start the container