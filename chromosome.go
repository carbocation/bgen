package bgen

// Chromosome takes the raw chromosome data and
// returns its standard string translation.
func Chromosome(chr uint16) string {
	chromosome := "NA"
	switch chr {
	case 1:
		chromosome = "01"
		break
	case 2:
		chromosome = "02"
		break
	case 3:
		chromosome = "03"
		break
	case 4:
		chromosome = "04"
		break
	case 5:
		chromosome = "05"
		break
	case 6:
		chromosome = "06"
		break
	case 7:
		chromosome = "07"
		break
	case 8:
		chromosome = "08"
		break
	case 9:
		chromosome = "09"
		break
	case 10:
		chromosome = "10"
		break
	case 11:
		chromosome = "11"
		break
	case 12:
		chromosome = "12"
		break
	case 13:
		chromosome = "13"
		break
	case 14:
		chromosome = "14"
		break
	case 15:
		chromosome = "15"
		break
	case 16:
		chromosome = "16"
		break
	case 17:
		chromosome = "17"
		break
	case 18:
		chromosome = "18"
		break
	case 19:
		chromosome = "19"
		break
	case 20:
		chromosome = "20"
		break
	case 21:
		chromosome = "21"
		break
	case 22:
		chromosome = "22"
		break
	case 23:
		chromosome = "0X"
		break
	case 24:
		chromosome = "0Y"
		break
	case 253:
		chromosome = "XY"
		break
	case 254:
		chromosome = "MT"
		break
	}

	return chromosome
}
