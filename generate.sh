#!/bin/bash

 protoc  --go_out=. --go-grpc_out=.  protopb/service.proto