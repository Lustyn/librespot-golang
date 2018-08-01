package player

import (
	"github.com/justync7/librespot-golang/src/Spotify"
	"bytes"
	"encoding/binary"
	"github.com/justync7/librespot-golang/src/librespot/connection"
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

// Read is an implementation of the io.Reader interface. Note that due to the nature of the streaming, we may return
// zero bytes when we are waiting for audio data from the Spotify servers, so make sure to wait for the io.EOF error
// before stopping playback.
func (i *ImageFile) Read(buf []byte) (int, error) {
	if i.Size() == 0 {
		return 0, nil
	}

	var err error
	if i.cursor >= i.Size() && i.eof {
		err = io.EOF
		return 0, err
	}

	length := min(i.Size() - i.cursor, len(buf))
	totalWritten := 0

	if length > 0 {
		//fmt.Printf("length: %d, buf len: %d", i.cursor+length, len(buf))
		writtenLen := copy(buf, i.data[i.cursor:i.cursor+length])
		totalWritten += writtenLen
		i.cursor += writtenLen
	}

	return totalWritten, err
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
		} else {
			i.eof = true
			break
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
