package protocol

import(
	"regexp"
	"os"
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"crypto/sha256"
	)

type FileReadProtocol struct {
	queue chan *Exchange
}

func MakeFileReadProtocol(threadCount int) *FileReadProtocol {
	p := &FileReadProtocol{
		make(chan *Exchange, threadCount),
	}
	for i := 0; i < threadCount; i++ {
		go p.runLoop()
	}
	return p
}

func (p *FileReadProtocol)Identifier() string {
	return "READ_FILE"
}

func (p *FileReadProtocol)Handle(request <-chan byte, response chan<- byte) <-chan StatusCode {
	done := make(chan StatusCode, 1)
	p.queue <- &Exchange{
		request,
		response,
		done,
	}
	return done
}

func (p *FileReadProtocol)runLoop() {
	for {
		rr := <- p.queue

		// Line 1 "READ_FILE:"
		r1, _ := regexp.Compile("\\A\\s*(\\S+)\\s*\\z")
		matches1 := r1.FindStringSubmatch(readLine(rr.request))
		if len(matches1) < 2 {
			respondError(ERROR_MALFORMED_REQUEST,rr.response)
			rr.done <- STATUS_ERROR
			continue
		}

		hash := sha256.New()
		hash.Write([]byte(matches1[1]))
		md := hash.Sum(nil)
		mdStr := hex.EncodeToString(md)

		fileName := "storage/"+mdStr
		file, ok1 := os.Open(fileName)

		if ok1 != nil {
			respondError(ERROR_FILE_NOT_FOUND,rr.response)
			rr.done <- STATUS_ERROR
			continue
		}

		send(rr.response,"CONTENT_BASE64:")
		writer := base64.NewEncoder(base64.StdEncoding,MakeChannelWriter(rr.response))

		reader := bufio.NewReader(file)
		b, err := reader.ReadByte()

		for err == nil {
			writer.Write([]byte{b})
			b, err = reader.ReadByte()
		}
		writer.Write([]byte{b})

		sendLine(rr.response,"")
		rr.done <- STATUS_SUCCESS_CONTINUE
	}
}