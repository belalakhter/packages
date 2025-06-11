**Arguments:**

### Address (string)
Address of endpoint ws://localhost:8080/ws or http://localhost:8080/sse
### Count (Int)
Initial Count to dispatch user first second
### Duration (int)
Duration to read from connection in seconds , for set 0 it will only send ping msg and do not write anything
### PumpCount (int)
Count for number of times clients should be doubled by each second for InitialCount 10 and pump count 4 It will:
first second : 10 users
second second: 20 users -- pump count 3
third second: 40 users  -- pump count 2
fourth second: 80 users -- pump count 1
Fifth second: 160 users --pump count 0
If pump count is set to zero then only initial users connection will be eastablished
### Type (string)
Test type "ws" for websocket or "sse" for sse


## Install
make install


## remove
make remove
