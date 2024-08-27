set shell := ["nu.exe", "-c"]

url :="https://example.com/"

working_url := "https://www.api.host-auto.ru/api/site/filters/params"

run: build
    rm -rf ./site_data
    .\timeit.exe -u {{ working_url }} -a 1

build:
    go build ./
