#!/usr/bin/env bash

test_description="Test http proxy over p2p"

. lib/test-lib.sh
WEB_SERVE_PORT=5099

function show_logs() {

    echo "*****************"
    echo "  RECEIVER LOG  " 
    echo "*****************"
    iptb logs 1
    echo "*****************"
    echo "  SENDER LOG  " 
    echo "*****************"
    iptb logs 0
    echo "*****************"
    echo "REMOTE_SERVER LOG" 
    echo $REMOTE_SERVER_LOG
    echo "*****************"
    cat $REMOTE_SERVER_LOG
}

function serve_http_once() {
    #
    # one shot http server (via nc) with static body
    #
    local body=$1
    local status_code=${2:-"200 OK"}
    local length=$((1 + ${#body}))
    REMOTE_SERVER_LOG="server.log"
    echo -e "HTTP/1.1 $status_code\nContent-length: $length\n\n$body" | nc -l $WEB_SERVE_PORT 2>&1 > $REMOTE_SERVER_LOG &  
    REMOTE_SERVER_PID=$!
}

function teardown_remote_server() {
    kill -9 $REMOTE_SERVER_PID > /dev/null 2>&1
    sleep 5
}

function curl_check_response_code() {
    local expected_status_code=$1
    local path_stub=${2:-http/$RECEIVER_ID/test/index.txt}
    local status_code=$(curl -s --write-out %{http_code} --output /dev/null $SENDER_URL/$path_stub)

    if [[ "$status_code" -ne "$expected_status_code" ]];
    then
        echo "Found status-code "$status_code", expected "$expected_status_code
        return 1
    fi

    return 0
}

function curl_send_proxy_request_and_check_response() {
    local expected_status_code=$1
    local expected_content=$2

    #
    # make a request to SENDER_IPFS via the proxy endpoint
    #
    CONTENT_PATH="retrieved-file"
    STATUS_CODE=$(curl -s -o $CONTENT_PATH --write-out %{http_code} $SENDER_URL/$RECEIVER_ID/test/index.txt)

    #
    # check status code
    #
    if [[ $STATUS_CODE -ne $expected_status_code ]];
    then
        echo -e "Found status-code "$STATUS_CODE", expected "$expected_status_code
        return 1
    fi

    #
    # check content
    #
    RESPONSE_CONTENT=$(tail -n 1 $CONTENT_PATH)
    if [[ "$RESPONSE_CONTENT" == "$expected_content" ]];
    then
        return 0
    else
        echo -e "Found response content:\n'"$RESPONSE_CONTENT"'\nthat differs from expected content:\n'"$expected_content"'"
        return 1
    fi
}

function curl_send_multipart_form_request() {
    local expected_status_code=$1
    local FILE_PATH="uploaded-file"
    FILE_CONTENT="curl will send a multipart-form POST request when sending a file which is handy"
    echo $FILE_CONTENT > $FILE_PATH
    #
    # send multipart form request
    #
    STATUS_CODE=$(curl -v -F file=@$FILE_PATH  $SENDER_URL/$RECEIVER_ID/test/index.txt)
    #
    # check status code
    #
    if [[ $STATUS_CODE -ne $expected_status_code ]];
    then
        echo -e "Found status-code "$STATUS_CODE", expected "$expected_status_code
        return 1
    fi
    #
    # check request method
    #
    if ! grep "POST /index.txt" $REMOTE_SERVER_LOG > /dev/null;
    then
        echo "Remote server request method/resource path was incorrect"
        show_logs
        return 1
    fi
    #
    # check request is multipart-form
    #
    if ! grep "Content-Type: multipart/form-data;" $REMOTE_SERVER_LOG > /dev/null;
    then
        echo "Request content-type was not multipart/form-data"
        show_logs
        return 1
    fi
    return 0
}

test_expect_success 'configure nodes' '
    iptb init -n 2 -p 0 -f --bootstrap=none &&
    ipfsi 0 config --json Experimental.Libp2pStreamMounting true &&
    ipfsi 1 config --json Experimental.Libp2pStreamMounting true &&
    ipfsi 0 config --json Experimental.P2pHttpProxy true
'

test_expect_success 'start and connect nodes' '
    iptb start && iptb connect 0 1
'

test_expect_success 'setup p2p listener on the receiver' '
    ipfsi 1 p2p listen --allow-custom-protocol test /ip4/127.0.0.1/tcp/$WEB_SERVE_PORT
'

test_expect_success 'setup environment variables' '
    RECEIVER_ID="$(iptb get id 1)" &&
    SENDER_URL="$(sed "s|^/ip[46]/\([^/]*\)/tcp/\([0-9]*\)\$|http://\\1:\\2/proxy/http|" < "$(iptb get path 0)/api")"
'

test_expect_success 'handle proxy http request propogates error response from remote' '
    serve_http_once "SORRY GUYS, I LOST IT" "404 Not Found" &&
    curl_send_proxy_request_and_check_response 404 "SORRY GUYS, I LOST IT"
'
teardown_remote_server

test_expect_success 'handle proxy http request sends bad-gateway when remote server not available ' '
    curl_send_proxy_request_and_check_response 502 ""
'

test_expect_success 'handle proxy http request ' '
    serve_http_once "THE WOODS ARE LOVELY DARK AND DEEP" &&
    curl_send_proxy_request_and_check_response 200 "THE WOODS ARE LOVELY DARK AND DEEP"
'
teardown_remote_server

test_expect_success 'handle proxy http request invalid request' '
    curl_check_response_code 400 DERPDERPDERP 
'

test_expect_success 'handle proxy http request unknown proxy peer ' '
    curl_check_response_code 502 unknown_peer/test/index.txt
'

test_expect_success 'handle multipart/form-data http request' '
    serve_http_once "OK" &&
    curl_send_multipart_form_request
'
teardown_remote_server

test_expect_success 'stop nodes' '
    iptb stop
'

test_done
