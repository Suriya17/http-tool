To build, just run 
    make
It generates the executable http-tool

To just send a HTTP and print output, do
    ./http-tool -url=<URL> [-https]
https is required to send HTTPS requests

To profile a webpage, do 
    ./http-tool -url=<URL> [-https] -profile=<NUM_OF_REQS>

The tool spawns a ton of go-workers to send HTTP requests. So, profiling with a large
number could lead to 429(Too many requests). It can easily be fixed by adding a small 
sleep timer to the go routine which I chose to ignore.

Response times to workers.dev are highly consistent even compared to popular websites 
like Google (it could be smaller page size too). But the variance in Response times 
seems to be smaller.
