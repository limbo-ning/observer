package instruction_test

import (
	"log"
	"testing"

	"obsessiontech/environment/environment/receiver/odor/instruction"
)

func TestParseInstruction(t *testing.T) {
	datagram := `POST /awdc.php HTTP/1.1s
HOST: 192.168.134.8
Content-Length:247
User-Agent: OdorCatch_v2.1
Content-Type: application/x-www-form-urlencoded
Connection: close
Accept: *.*

stationid=005&date=20190615&time=1037&ou=00004019&signal=014613&H2S=001430&NH3=014078&VOC=013107&M01=004019&M02=-005238&P01=000001&E01=-000389&E02=-000134&intemp= 389&outtemp= 318&outhumidity=507&winddir=219&windspeed=006&atm=000&heater=0&cooler=0`

	i, err := instruction.Parse(datagram)
	if err != nil {
		panic(err)
	}

	log.Println(i)
}
