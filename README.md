[![Go Report Card](https://goreportcard.com/badge/logrange/logrange)](https://goreportcard.com/report/logrange/logrange) [![Build Status](https://travis-ci.org/logrange/logrange.svg?branch=master)](https://travis-ci.org/logrange/logrange) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/logrange/logrange/blob/master/LICENSE)

# Logrange - streaming database 
Logrange is highly performant streaming database, which allows to aggregate, structuring and persisting streams of records like application logs, system metrics, audit logs etc. Logrange persists input streams on the disk and provides an API for accessing the stored data.

## What exactly does that mean?
Modern systems can consist of thousands of different sub-systems and applications, which usually write information about their activity into a file or application log. The application log is a stream of records, where the records appear in the order of writing them into the log.

Logrange allows to collect the logs in a very efficient manner and store the log records on the disk for further processing.

The data, which can be stored in the database is not limited by the application logs only. For example, other streaming data like System metrics, autdit logs, application events could be stored into Logrange as well.

## What does Logrange allow to do?
Logrange does the following things: 
* Collecting streams of records in different formats from multiple sources 
* Accessing to the aggregated data via API, which allows searching, merging, and filtering data from different streams of records.
* Forwarding filtered or all aggregated data to 3rd party systems.

## What about other log aggregation solutions? How Logrange is different?
Logrange is intended for storing thousands of streams of records, like application logs, allowing millions writes per second. The disk structures Logrange uses scale well, so its performance doesn't depend on how big the stored data is - either it is megabytes or terabytes of the data.

Logrange is focused on streams processing, but not on the data indexing. It is not indended for full text search, even though we do support features like `search` in Logrange as well. Logrange is optimized to work with streams of records and big arrays of the log data.

Moreover, Logrange allows to store not only application logs, but any streaming data, which could be collected from 3rd party system. This makes Logrange an integration tool of different types of streams collected from different sources and stored in one databas sutable for furhter processing.

The features like analytics, statistics and data learning could be easily built on top of Logrange database.

# Introduction
Logrange database be run as stand-alone application or as a cluster (distributed system which consists of multiple instances). It provides an API which can be used for writing by _Collectors_ - software clients which `writes` input streams of records into the Logrange database. Another type of clients are _Consumers_ that use Logrange API for retrieving data and sending it to another system for further processing or just show it to a user in interactive manner:

![Logrange Structure](https://raw.githubusercontent.com/logrange/logrange/master/doc/pics/Logrange%20Structure.png)

## Data structures
Logrange recognizes the following entities:
* _stream_ - a sequence of _records_. Every stream contains zero or a natural number of records.
* _source_ - is a input stream of _records_ which is written by one or many collectors
* _tags_ - is a combination of key-value pairs, applied to a _source_
* _record_ - is an atomic piece of information from a _stream_. Every _record_ contains 2+fields.
* _field_ - is a key-value pair, which is part of a record.
### Sources and tags
In Logrange every persisted stream of records is recoginized as a _source_. Every _source_ has an unique combination of _tags_. _tags_ are a comma separated key-value pairs written in the form like:
```
name=application1,ip="127.0.0.1"
```
To address a stream for `write` operation an unique combintaion of _tags_ must be provided. For example, Collector, when writes records for as stream, must provide _tags_ combination that idenfies the source uniqueuly. 

To select one or more sources the condition of tags should be provided. For example:
* `name=application1,ip="127.0.0.1"` - select ALL sources which tags contain `name=application1` and `ip="127.0.0.1"`
* `name=application1 OR ip="127.0.0.1"` - selects all sources which tags contain either `name=application1` pair, or `ip="127.0.0.1"`pair, or both of them
* `name LIKE 'app*'` - selects all sources which tags contain key-value pair with the key="name" and the value which starts from "app"
etc.

### Records and fields
A _stream_ consists of ordered _records_. Every record contains 2 mandatory fields and 0 or more optional, custom fields. The mandatory fields are:
* `ts` - the records timestamp. It is set by Collector and it can be 0
* `msg` - the record content. This is just a slice of bytes which can be treated as a text.
Optional fields are key-value pairs, which value can be addressed by the field name with `fields:` pfrefix. Fields can be combined to expressions. For example:
* `msg contains "abc"` - matches records, for which `msg` field contains text "abc"
* `msg contains "ERROR" AND fields:name = "app1"` - matches records, for which `msg` field contains text "ERROR" AND the field with the key="name" has value="app1"
etc.

## Main components
### Aggregator
### Clients
#### Log Collector
#### CLI tool
#### Log Forwarder
## Logrange Query Language (LQL)
# Available Configurations
# Roadmap

