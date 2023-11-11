# Microservices Example

## Quickstart

`docker compose up`

## Services

**Accounts**

Accounts service is responsible for `Users` and associated `Teams`.

**Locations**

`Locations` belong to a `Team` and emit data points collected from mock IoT devices in the field. These data points are sent to a Kafka instance to be picked up by other services like the Readings service.

**Readings**

The Readings service pulls data from the Kafka queue to record location KPIs. These data can then be processed to produce reports.

