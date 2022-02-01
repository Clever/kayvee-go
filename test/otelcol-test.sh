#!/bin/bash

# start otel colllector container
echo "Starting opentelemetry Docker contianers..."
docker run -d --rm -p 13133:13133 -p 14250:14250 -p 14268:14268 -p 55678-55679:55678-55679 -p 4317:4317 -p 8888:8888 -p 9411:9411 -v "${PWD}/test/otel-config.yaml":/otel-config.yaml --name otelcol-test otel/opentelemetry-collector --config otel-config.yaml
if [ $? -eq 0 ]; then
    echo "Container started..."
else
    exit 1
fi

# wair for the collector to start
printf "Waiting for otelcol to start..."
while true ; do 
    printf "."
    result=$(docker logs otelcol-test 2>&1 | grep 'Everything is ready' | wc -l)
    if [ "$result" -eq 1 ] ; then
        echo ""
        echo "Started..."
        break
    fi
    sleep 1
done

# run tests
OTEL_TEST=test OTEL_COLLECTOR_URL=tcp://localhost:4317 go test -count=1 -v -run TestLogOTLCounter ./logger
OTEL_TEST=test OTEL_COLLECTOR_URL=tcp://localhost:4317 go test -count=1 -v -run TestLogOTLGaugeInt ./logger
OTEL_TEST=test OTEL_COLLECTOR_URL=tcp://localhost:4317 go test -count=1 -v -run TestLogOTLGaugeFloat ./logger
if [ $? -ne 0 ]; then
    echo "Go tests failed..."
fi
echo "Go tests successfully ran..."

# print the containers logs
docker logs otelcol-test

# check the logs for the metrics
names=( "Name: testlogcounter" "Name: testloggaugeint" "Name: testloggaugefloat")
for name in "${names[@]}"; do
    printf "Checking for $name"
    result=$(docker logs otelcol-test 2>&1 | grep "$name" | wc -l)
    if [ "$result" -eq 1 ] ; then
        echo " -> Found [OK]"
    else
        echo " -> Not found"
        exit 1
    fi
done
values=( "Value: 1" "Value: 2" "Value: 0" "Value: 4" "Value: 4.000000" "Value: 0.000000")
for value in "${values[@]}"; do
    printf "Checking for $value"
    result=$(docker logs otelcol-test 2>&1 | grep "$value" | wc -l)
    if [ "$result" -gt 0 ] ; then
        echo " -> Found [OK]"
    else
        echo " -> Not found"
        exit 1
    fi
done
labels=( "key1: STRING(val1)" "key2: STRING(val2)" )
for label in "${labels[@]}"; do
    printf "Checking for $label"
    result=$(docker logs otelcol-test 2>&1 | grep "$label" | wc -l)
    if [ "$result" -gt 1 ] ; then
        echo " -> Found [OK]"
    else
        echo " -> Not found"
        exit 1
    fi
done
echo "PASS"



# stop test container
echo "Stopping docker container"
docker stop otelcol-test