package player

import (
	"github.com/lustyn/librespot-golang/src/Spotify"
	"bytes"
	"encoding/binary"
	"github.com/lustyn/librespot-golang/src/librespot/connection"
	"io"
	"fmt"
)

type ImageFile struct {
	format         Spotify.Image_Size
	fileId         []byte
	player         *Player
	responseChan   chan []byte
	data           []byte
	cursor         int
	eof            bool
}

func newImageFile(file *Spotify.Image, player *Player) *ImageFile {
	return newImageFileWithIdAndFormat(file.GetFileId(), *file.Size, player)
}

func newImageFileWithIdAndFormat(fileId []byte, format Spotify.Image_Size, player *Player) *ImageFile {
	return &ImageFile{
		player:        player,
		fileId:        fileId,
		format:        format,
		responseChan:  make(chan []byte),
		eof:           false,
	}
}

// Size returns the size, in bytes, of the final audio file
func (i *ImageFile) Size() int {
	return len(i.data)
}

func (i *ImageFile) IsEOF() bool {
	return i.eof
}

// Read is an implementation of the io.Reader interface. Note that due to the nature of the streaming, we may return
// zero bytes when we are waiting for audio data from the Spotify servers, so make sure to wait for the io.EOF error
// before stopping playback.
func (i *ImageFile) Read(buf []byte) (int, error) {
	if i.Size() == 0 || buf == nil || i.data == nil {
		return 0, nil
	}

	if i.cursor >= i.Size() && i.eof {
		return 0, io.EOF
	}/* else if i.cursor >= i.Size() {
		return 0, nil
	}*/
	
	totalWritten := 0
	
	writtenLen := copy(buf, i.data[i.cursor:])
	
	totalWritten += writtenLen
	i.cursor += writtenLen

	return totalWritten, nil
}

func buildCoverArtRequest(channel uint16, fileId []byte) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, channel)
	binary.Write(buf, binary.BigEndian, uint16(0x0))
	buf.Write(fileId)

	return buf.Bytes()
}

func (i *ImageFile) loadData() {
	go i.load()
}

func (i *ImageFile) load() error {
	channel := i.player.AllocateChannel()
	channel.onData = i.onChannelData
	channel.onHeader = i.onChannelHeader
	channel.onRelease = i.onRelease

	err := i.player.stream.SendPacket(connection.PacketImage, buildCoverArtRequest(channel.num, i.fileId))

	if err != nil {
		return err
	}

	for {
		chunk := <-i.responseChan
		chunkLen := len(chunk)
		//fmt.Printf("chunkLen: %d\n", chunkLen)
		if chunkLen > 0 {
			i.data = append(i.data, chunk...)

			// fmt.Printf("Read %d/%d of chunk %d\n", sz, expSize, i)
		} else if i.eof {
			return io.EOF
		} else {
			//i.eof = true
			fmt.Printf("If this freezes something is wrong...\n", chunkLen)
			//break
		}
	}

	return nil

}

func (i *ImageFile) onChannelHeader(channel *Channel, id byte, data *bytes.Reader) uint16 {
	read := uint16(0)

	fmt.Printf("Got header id: 0x%x",id)

	return read
}

func (i *ImageFile) onChannelData(channel *Channel, data []byte) uint16 {
	if data != nil {
		i.responseChan <- data

		return 0
	} else {
		i.responseChan <- []byte{}

		return 0
	}

}

func (i *ImageFile) onRelease(channel *Channel) {
	i.eof = true
}
