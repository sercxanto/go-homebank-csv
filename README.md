# go-homebank-csv

![Build and test](https://github.com/sercxanto/go-homebank-csv/actions/workflows/build-and-test.yml/badge.svg)
![golangci-lint](https://github.com/sercxanto/go-homebank-csv/actions/workflows/golangci-lint.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/sercxanto/go-homebank-csv.svg)](https://pkg.go.dev/github.com/sercxanto/go-homebank-csv)
[![codecov](https://codecov.io/gh/sercxanto/go-homebank-csv/graph/badge.svg?token=HB6HHXV7X6)](https://codecov.io/gh/sercxanto/go-homebank-csv)

This tool converts "comma separated values" (CSV) files from several banks to the
["Transaction import CSV format"](http://homebank.free.fr/help/misc-csvformat.html)
supported by [HomeBank](http://homebank.free.fr/).

HomeBank is a crossplatform free and easy accounting software.

## Supported formats

* MoneyWallet
    * [MoneyWallet](https://f-droid.org/en/packages/com.oriondev.moneywallet) is an expense manager for Android.
go-homebank-csv supports parsing and converting the CSV export format.
* Barclaycard
    * Not exactly CSV, this is the excel export format of Barclays VISA card as found on [www.barclays.de](https://www.barclays.de).
* Volksbank
    * This is the CSV export format used by a German Volksbank. Most probably all Volksbanks have the same format.
* Comdirect
    * This is the giro account CSV export format used by [www.comdirect.de](https://www.comdirect.de).
It has some weird encoding and the internal structure changes often.
* DKB
    * This is the giro account CSV export format used by [www.dkb.de](https://www.dkb.de).

## Usage

List supported formats:

```shell
go-homebank-csv list-formats
```

### Convert single files

Convert one file using format autodetection:

```shell
go-homebank-csv convert input-file.csv output-file.csv
```

Convert one file using known format:

```shell
go-homebank-csv convert --format=MoneyWallet input-file.csv output-file.csv
```

### Batch convert a folder of files

You can autoconvert a defined set of folders. To use this feature a config file is needed.

The config file is expected at the following location:

```text
$XDG_CONFIG_HOME/go-homebank-csv/config.yml
```

For the different operating system this is usually at the following places:

* Linux: `~/.config/go-homebank-csv/config.yml`
* MacOS: `~/Library/Application Support/go-homebank-csv/config.yml`
* Windows: `"LocalAppData"/go-homebank-csv/config.yml`

#### Config file format

A minimal version of a config file looks like the following:

```yaml
batchconvert:
  sets:
  - name: Bank 1
    inputdir: /home/user/finance/barclaycard/xlsx
    outputdir: /home/user/finance/barclaycard/homebankcsv
```

The fields have the following meaning:

* `name`: The name of the entry. The name must be unique.
* `inputdir`: Where to search for files (non recursively).
* `outputdir`: Where to place the converted files.

The minimal version can be amended by optional settings:

```yaml
batchconvert:
  sets:
  - name: Bank 1
    inputdir: /home/user/finance/barclaycard/xlsx
    outputdir: /home/user/finance/barclaycard/homebankcsv
    fileglobpattern: "*.xlsx"
    filemaxagedays: 3
    format: Barclaycard
```

The additional fields have the following meaning:

* `fileglobpattern`: Narrow down the files to search for in `inputdir` by this pattern.
   The glob pattern follows the one from the package [path/filepath](https://pkg.go.dev/path/filepath#Match)
   from golang standard library.
* `filemaxagedays`: Narrow down the files to search for in `inputdir` by specifying a maximum age in days
   (modification timestamp) in days. Only positive numbers are allowed.
* `format`: Specify the exact format to be expected. If not given an probably error-prone and time-consuming
   autodetection is done.

#### Command line example

With a config file like this:

```yaml
batchconvert:
  sets:
  - name: Bank 1
    inputdir: /home/user/finance/barclaycard/xlsx
    outputdir: /home/user/finance/barclaycard/homebankcsv
    fileglobpattern: "*.xlsx"
    filemaxagedays: 3
    format: Barclaycard
  - name: Bank 2
    inputdir: /home/user/finance/volksbank/csv
    outputdir: /home/user/finance/volksbank/homebankcsv
    fileglobpattern: "*.csv"
    filemaxagedays: 2
    format: Volksbank
```

Call the sub-command `batchconvert` like this:

```shell
go-homebank-csv batchconvert
```

`go-homebank-csv` will do the following:

* Search in directory "/home/user/finance/barclaycard/xlsx" for files matching "*.xlsx"
  which have been modified not longer ago than 3 days
* Check if a file with the same basename is already at "/home/user/finance/barclaycard/homebankcsv"
* If this is not the case convert the found files using the same base name with an extention ".csv"
  and store them at "/home/user/finance/barclaycard/homebankcsv"
* Search in directory "/home/user/finance/volksbank/csv" for files matching "*.csv"
  which have been modified not longer ago than 2 days
* Check if a file with the same basename is already at "/home/user/finance/volksbank/homebankcsv"
* If this is not the case convert the found files using the same base name with an extention ".csv"
  and store them at "/home/user/finance/volksbank/homebankcsv"
