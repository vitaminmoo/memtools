package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/vitaminmoo/memtools/hexpat/runtime"
)

type EM uint16

const (
	EMEMNONE          EM = 0
	EMEMM32           EM = 1
	EMEMSPARC         EM = 2
	EMEM386           EM = 3
	EMEM68K           EM = 4
	EMEM88K           EM = 5
	EMEMIAMCU         EM = 6
	EMEM860           EM = 7
	EMEMMIPS          EM = 8
	EMEMS370          EM = 9
	EMEMMIPSRS4BE     EM = 10
	EMEMPARISC        EM = 15
	EMEMVPP500        EM = 17
	EMEMSPARC32PLUS   EM = 18
	EMEM960           EM = 19
	EMEMPPC           EM = 20
	EMEMPPC64         EM = 21
	EMEMS390          EM = 22
	EMEMSPU           EM = 23
	EMEMV800          EM = 36
	EMEMFR20          EM = 37
	EMEMRH32          EM = 38
	EMEMRCE           EM = 39
	EMEMARM           EM = 40
	EMEMALPHA         EM = 41
	EMEMSH            EM = 42
	EMEMSPARCV9       EM = 43
	EMEMTRICORE       EM = 44
	EMEMARC           EM = 45
	EMEMH8300         EM = 46
	EMEMH8300H        EM = 47
	EMEMH8S           EM = 48
	EMEMH8500         EM = 49
	EMEMIA64          EM = 50
	EMEMMIPSX         EM = 51
	EMEMCOLDFIRE      EM = 52
	EMEM68HC12        EM = 53
	EMEMMMA           EM = 54
	EMEMPCP           EM = 55
	EMEMNCPU          EM = 56
	EMEMNDR1          EM = 57
	EMEMSTARCORE      EM = 58
	EMEMME16          EM = 59
	EMEMST100         EM = 60
	EMEMTINYJ         EM = 61
	EMEMX8664         EM = 62
	EMEMPDSP          EM = 63
	EMEMPDP10         EM = 64
	EMEMPDP11         EM = 65
	EMEMFX66          EM = 66
	EMEMST9PLUS       EM = 67
	EMEMST7           EM = 68
	EMEM68HC16        EM = 69
	EMEM68HC11        EM = 70
	EMEM68HC08        EM = 71
	EMEM68HC05        EM = 72
	EMEMSVX           EM = 73
	EMEMST19          EM = 74
	EMEMVAX           EM = 75
	EMEMCRIS          EM = 76
	EMEMJAVELIN       EM = 77
	EMEMFIREPATH      EM = 78
	EMEMZSP           EM = 79
	EMEMMMIX          EM = 80
	EMEMHUANY         EM = 81
	EMEMPRISM         EM = 82
	EMEMAVR           EM = 83
	EMEMFR30          EM = 84
	EMEMD10V          EM = 85
	EMEMD30V          EM = 86
	EMEMV850          EM = 87
	EMEMM32R          EM = 88
	EMEMMN10300       EM = 89
	EMEMMN10200       EM = 90
	EMEMPJ            EM = 91
	EMEMOPENRISC      EM = 92
	EMEMARCCOMPACT    EM = 93
	EMEMXTENSA        EM = 94
	EMEMVIDEOCORE     EM = 95
	EMEMTMMGPP        EM = 96
	EMEMNS32K         EM = 97
	EMEMTPC           EM = 98
	EMEMSNP1K         EM = 99
	EMEMST200         EM = 100
	EMEMIP2K          EM = 101
	EMEMMAX           EM = 102
	EMEMCR            EM = 103
	EMEMF2MC16        EM = 104
	EMEMMSP430        EM = 105
	EMEMBLACKFIN      EM = 106
	EMEMSEC33         EM = 107
	EMEMSEP           EM = 108
	EMEMARCA          EM = 109
	EMEMUNICORE       EM = 110
	EMEMEXCESS        EM = 111
	EMEMDXP           EM = 112
	EMEMALTERANIOS2   EM = 113
	EMEMCRX           EM = 114
	EMEMXGATE         EM = 115
	EMEMC166          EM = 116
	EMEMM16C          EM = 117
	EMEMDSPIC30F      EM = 118
	EMEMCE            EM = 119
	EMEMM32C          EM = 120
	EMEMTSK3000       EM = 131
	EMEMRS08          EM = 132
	EMEMSHARC         EM = 133
	EMEMECOG2         EM = 134
	EMEMSCORE7        EM = 135
	EMEMDSP24         EM = 136
	EMEMVIDEOCORE3    EM = 137
	EMEMLATTICEMICO32 EM = 138
	EMEMSEC17         EM = 139
	EMEMTIC6000       EM = 140
	EMEMTIC2000       EM = 141
	EMEMTIC5500       EM = 142
	EMEMTIARP32       EM = 143
	EMEMTIPRU         EM = 144
	EMEMMMDSPPLUS     EM = 160
	EMEMCYPRESSM8C    EM = 161
	EMEMR32C          EM = 162
	EMEMTRIMEDIA      EM = 163
	EMEMQDSP6         EM = 164
	EMEM8051          EM = 165
	EMEMSTXP7X        EM = 166
	EMEMNDS32         EM = 167
	EMEMECOG1         EM = 168
	EMEMECOG1X        EM = 168
	EMEMMAXQ30        EM = 169
	EMEMXIMO16        EM = 170
	EMEMMANIK         EM = 171
	EMEMCRAYNV2       EM = 172
	EMEMRX            EM = 173
	EMEMMETAG         EM = 174
	EMEMMCSTELBRUS    EM = 175
	EMEMECOG16        EM = 176
	EMEMCR16          EM = 177
	EMEMETPU          EM = 178
	EMEMSLE9X         EM = 179
	EMEML10M          EM = 180
	EMEMK10M          EM = 181
	EMEMAARCH64       EM = 183
	EMEMAVR32         EM = 185
	EMEMSTM8          EM = 186
	EMEMTILE64        EM = 187
	EMEMTILEPRO       EM = 188
	EMEMMICROBLAZE    EM = 189
	EMEMCUDA          EM = 190
	EMEMTILEGX        EM = 191
	EMEMCLOUDSHIELD   EM = 192
	EMEMCOREA1ST      EM = 193
	EMEMCOREA2ND      EM = 194
	EMEMARCCOMPACT2   EM = 195
	EMEMOPEN8         EM = 196
	EMEMRL78          EM = 197
	EMEMVIDEOCORE5    EM = 198
	EMEM78KOR         EM = 199
	EMEM56800EX       EM = 200
	EMEMBA1           EM = 201
	EMEMBA2           EM = 202
	EMEMXCORE         EM = 203
	EMEMMCHPPIC       EM = 204
	EMEMINTEL205      EM = 205
	EMEMINTEL206      EM = 206
	EMEMINTEL207      EM = 207
	EMEMINTEL208      EM = 208
	EMEMINTEL209      EM = 209
	EMEMKM32          EM = 210
	EMEMKMX32         EM = 211
	EMEMKMX16         EM = 212
	EMEMKMX8          EM = 213
	EMEMKVARC         EM = 214
	EMEMCDP           EM = 215
	EMEMCOGE          EM = 216
	EMEMCOOL          EM = 217
	EMEMNORC          EM = 218
	EMEMCSRKALIMBA    EM = 219
	EMEMZ80           EM = 220
	EMEMVISIUM        EM = 221
	EMEMFT32          EM = 222
	EMEMMOXIE         EM = 223
	EMEMAMDGPU        EM = 224
	EMEMRISCV         EM = 243
)

func (e EM) String() string {
	switch e {
	case EMEM386:
		return fmt.Sprintf("EM386 (%d)", uint16(e))
	case EMEM56800EX:
		return fmt.Sprintf("EM56800EX (%d)", uint16(e))
	case EMEM68HC05:
		return fmt.Sprintf("EM68HC05 (%d)", uint16(e))
	case EMEM68HC08:
		return fmt.Sprintf("EM68HC08 (%d)", uint16(e))
	case EMEM68HC11:
		return fmt.Sprintf("EM68HC11 (%d)", uint16(e))
	case EMEM68HC12:
		return fmt.Sprintf("EM68HC12 (%d)", uint16(e))
	case EMEM68HC16:
		return fmt.Sprintf("EM68HC16 (%d)", uint16(e))
	case EMEM68K:
		return fmt.Sprintf("EM68K (%d)", uint16(e))
	case EMEM78KOR:
		return fmt.Sprintf("EM78KOR (%d)", uint16(e))
	case EMEM8051:
		return fmt.Sprintf("EM8051 (%d)", uint16(e))
	case EMEM860:
		return fmt.Sprintf("EM860 (%d)", uint16(e))
	case EMEM88K:
		return fmt.Sprintf("EM88K (%d)", uint16(e))
	case EMEM960:
		return fmt.Sprintf("EM960 (%d)", uint16(e))
	case EMEMAARCH64:
		return fmt.Sprintf("EMAARCH64 (%d)", uint16(e))
	case EMEMALPHA:
		return fmt.Sprintf("EMALPHA (%d)", uint16(e))
	case EMEMALTERANIOS2:
		return fmt.Sprintf("EMALTERANIOS2 (%d)", uint16(e))
	case EMEMAMDGPU:
		return fmt.Sprintf("EMAMDGPU (%d)", uint16(e))
	case EMEMARC:
		return fmt.Sprintf("EMARC (%d)", uint16(e))
	case EMEMARCA:
		return fmt.Sprintf("EMARCA (%d)", uint16(e))
	case EMEMARCCOMPACT:
		return fmt.Sprintf("EMARCCOMPACT (%d)", uint16(e))
	case EMEMARCCOMPACT2:
		return fmt.Sprintf("EMARCCOMPACT2 (%d)", uint16(e))
	case EMEMARM:
		return fmt.Sprintf("EMARM (%d)", uint16(e))
	case EMEMAVR:
		return fmt.Sprintf("EMAVR (%d)", uint16(e))
	case EMEMAVR32:
		return fmt.Sprintf("EMAVR32 (%d)", uint16(e))
	case EMEMBA1:
		return fmt.Sprintf("EMBA1 (%d)", uint16(e))
	case EMEMBA2:
		return fmt.Sprintf("EMBA2 (%d)", uint16(e))
	case EMEMBLACKFIN:
		return fmt.Sprintf("EMBLACKFIN (%d)", uint16(e))
	case EMEMC166:
		return fmt.Sprintf("EMC166 (%d)", uint16(e))
	case EMEMCDP:
		return fmt.Sprintf("EMCDP (%d)", uint16(e))
	case EMEMCE:
		return fmt.Sprintf("EMCE (%d)", uint16(e))
	case EMEMCLOUDSHIELD:
		return fmt.Sprintf("EMCLOUDSHIELD (%d)", uint16(e))
	case EMEMCOGE:
		return fmt.Sprintf("EMCOGE (%d)", uint16(e))
	case EMEMCOLDFIRE:
		return fmt.Sprintf("EMCOLDFIRE (%d)", uint16(e))
	case EMEMCOOL:
		return fmt.Sprintf("EMCOOL (%d)", uint16(e))
	case EMEMCOREA1ST:
		return fmt.Sprintf("EMCOREA1ST (%d)", uint16(e))
	case EMEMCOREA2ND:
		return fmt.Sprintf("EMCOREA2ND (%d)", uint16(e))
	case EMEMCR:
		return fmt.Sprintf("EMCR (%d)", uint16(e))
	case EMEMCR16:
		return fmt.Sprintf("EMCR16 (%d)", uint16(e))
	case EMEMCRAYNV2:
		return fmt.Sprintf("EMCRAYNV2 (%d)", uint16(e))
	case EMEMCRIS:
		return fmt.Sprintf("EMCRIS (%d)", uint16(e))
	case EMEMCRX:
		return fmt.Sprintf("EMCRX (%d)", uint16(e))
	case EMEMCSRKALIMBA:
		return fmt.Sprintf("EMCSRKALIMBA (%d)", uint16(e))
	case EMEMCUDA:
		return fmt.Sprintf("EMCUDA (%d)", uint16(e))
	case EMEMCYPRESSM8C:
		return fmt.Sprintf("EMCYPRESSM8C (%d)", uint16(e))
	case EMEMD10V:
		return fmt.Sprintf("EMD10V (%d)", uint16(e))
	case EMEMD30V:
		return fmt.Sprintf("EMD30V (%d)", uint16(e))
	case EMEMDSP24:
		return fmt.Sprintf("EMDSP24 (%d)", uint16(e))
	case EMEMDSPIC30F:
		return fmt.Sprintf("EMDSPIC30F (%d)", uint16(e))
	case EMEMDXP:
		return fmt.Sprintf("EMDXP (%d)", uint16(e))
	case EMEMECOG1:
		return fmt.Sprintf("EMECOG1 (%d)", uint16(e))
	case EMEMECOG16:
		return fmt.Sprintf("EMECOG16 (%d)", uint16(e))
	case EMEMECOG2:
		return fmt.Sprintf("EMECOG2 (%d)", uint16(e))
	case EMEMETPU:
		return fmt.Sprintf("EMETPU (%d)", uint16(e))
	case EMEMEXCESS:
		return fmt.Sprintf("EMEXCESS (%d)", uint16(e))
	case EMEMF2MC16:
		return fmt.Sprintf("EMF2MC16 (%d)", uint16(e))
	case EMEMFIREPATH:
		return fmt.Sprintf("EMFIREPATH (%d)", uint16(e))
	case EMEMFR20:
		return fmt.Sprintf("EMFR20 (%d)", uint16(e))
	case EMEMFR30:
		return fmt.Sprintf("EMFR30 (%d)", uint16(e))
	case EMEMFT32:
		return fmt.Sprintf("EMFT32 (%d)", uint16(e))
	case EMEMFX66:
		return fmt.Sprintf("EMFX66 (%d)", uint16(e))
	case EMEMH8300:
		return fmt.Sprintf("EMH8300 (%d)", uint16(e))
	case EMEMH8300H:
		return fmt.Sprintf("EMH8300H (%d)", uint16(e))
	case EMEMH8500:
		return fmt.Sprintf("EMH8500 (%d)", uint16(e))
	case EMEMH8S:
		return fmt.Sprintf("EMH8S (%d)", uint16(e))
	case EMEMHUANY:
		return fmt.Sprintf("EMHUANY (%d)", uint16(e))
	case EMEMIA64:
		return fmt.Sprintf("EMIA64 (%d)", uint16(e))
	case EMEMIAMCU:
		return fmt.Sprintf("EMIAMCU (%d)", uint16(e))
	case EMEMINTEL205:
		return fmt.Sprintf("EMINTEL205 (%d)", uint16(e))
	case EMEMINTEL206:
		return fmt.Sprintf("EMINTEL206 (%d)", uint16(e))
	case EMEMINTEL207:
		return fmt.Sprintf("EMINTEL207 (%d)", uint16(e))
	case EMEMINTEL208:
		return fmt.Sprintf("EMINTEL208 (%d)", uint16(e))
	case EMEMINTEL209:
		return fmt.Sprintf("EMINTEL209 (%d)", uint16(e))
	case EMEMIP2K:
		return fmt.Sprintf("EMIP2K (%d)", uint16(e))
	case EMEMJAVELIN:
		return fmt.Sprintf("EMJAVELIN (%d)", uint16(e))
	case EMEMK10M:
		return fmt.Sprintf("EMK10M (%d)", uint16(e))
	case EMEMKM32:
		return fmt.Sprintf("EMKM32 (%d)", uint16(e))
	case EMEMKMX16:
		return fmt.Sprintf("EMKMX16 (%d)", uint16(e))
	case EMEMKMX32:
		return fmt.Sprintf("EMKMX32 (%d)", uint16(e))
	case EMEMKMX8:
		return fmt.Sprintf("EMKMX8 (%d)", uint16(e))
	case EMEMKVARC:
		return fmt.Sprintf("EMKVARC (%d)", uint16(e))
	case EMEML10M:
		return fmt.Sprintf("EML10M (%d)", uint16(e))
	case EMEMLATTICEMICO32:
		return fmt.Sprintf("EMLATTICEMICO32 (%d)", uint16(e))
	case EMEMM16C:
		return fmt.Sprintf("EMM16C (%d)", uint16(e))
	case EMEMM32:
		return fmt.Sprintf("EMM32 (%d)", uint16(e))
	case EMEMM32C:
		return fmt.Sprintf("EMM32C (%d)", uint16(e))
	case EMEMM32R:
		return fmt.Sprintf("EMM32R (%d)", uint16(e))
	case EMEMMANIK:
		return fmt.Sprintf("EMMANIK (%d)", uint16(e))
	case EMEMMAX:
		return fmt.Sprintf("EMMAX (%d)", uint16(e))
	case EMEMMAXQ30:
		return fmt.Sprintf("EMMAXQ30 (%d)", uint16(e))
	case EMEMMCHPPIC:
		return fmt.Sprintf("EMMCHPPIC (%d)", uint16(e))
	case EMEMMCSTELBRUS:
		return fmt.Sprintf("EMMCSTELBRUS (%d)", uint16(e))
	case EMEMME16:
		return fmt.Sprintf("EMME16 (%d)", uint16(e))
	case EMEMMETAG:
		return fmt.Sprintf("EMMETAG (%d)", uint16(e))
	case EMEMMICROBLAZE:
		return fmt.Sprintf("EMMICROBLAZE (%d)", uint16(e))
	case EMEMMIPS:
		return fmt.Sprintf("EMMIPS (%d)", uint16(e))
	case EMEMMIPSRS4BE:
		return fmt.Sprintf("EMMIPSRS4BE (%d)", uint16(e))
	case EMEMMIPSX:
		return fmt.Sprintf("EMMIPSX (%d)", uint16(e))
	case EMEMMMA:
		return fmt.Sprintf("EMMMA (%d)", uint16(e))
	case EMEMMMDSPPLUS:
		return fmt.Sprintf("EMMMDSPPLUS (%d)", uint16(e))
	case EMEMMMIX:
		return fmt.Sprintf("EMMMIX (%d)", uint16(e))
	case EMEMMN10200:
		return fmt.Sprintf("EMMN10200 (%d)", uint16(e))
	case EMEMMN10300:
		return fmt.Sprintf("EMMN10300 (%d)", uint16(e))
	case EMEMMOXIE:
		return fmt.Sprintf("EMMOXIE (%d)", uint16(e))
	case EMEMMSP430:
		return fmt.Sprintf("EMMSP430 (%d)", uint16(e))
	case EMEMNCPU:
		return fmt.Sprintf("EMNCPU (%d)", uint16(e))
	case EMEMNDR1:
		return fmt.Sprintf("EMNDR1 (%d)", uint16(e))
	case EMEMNDS32:
		return fmt.Sprintf("EMNDS32 (%d)", uint16(e))
	case EMEMNONE:
		return fmt.Sprintf("EMNONE (%d)", uint16(e))
	case EMEMNORC:
		return fmt.Sprintf("EMNORC (%d)", uint16(e))
	case EMEMNS32K:
		return fmt.Sprintf("EMNS32K (%d)", uint16(e))
	case EMEMOPEN8:
		return fmt.Sprintf("EMOPEN8 (%d)", uint16(e))
	case EMEMOPENRISC:
		return fmt.Sprintf("EMOPENRISC (%d)", uint16(e))
	case EMEMPARISC:
		return fmt.Sprintf("EMPARISC (%d)", uint16(e))
	case EMEMPCP:
		return fmt.Sprintf("EMPCP (%d)", uint16(e))
	case EMEMPDP10:
		return fmt.Sprintf("EMPDP10 (%d)", uint16(e))
	case EMEMPDP11:
		return fmt.Sprintf("EMPDP11 (%d)", uint16(e))
	case EMEMPDSP:
		return fmt.Sprintf("EMPDSP (%d)", uint16(e))
	case EMEMPJ:
		return fmt.Sprintf("EMPJ (%d)", uint16(e))
	case EMEMPPC:
		return fmt.Sprintf("EMPPC (%d)", uint16(e))
	case EMEMPPC64:
		return fmt.Sprintf("EMPPC64 (%d)", uint16(e))
	case EMEMPRISM:
		return fmt.Sprintf("EMPRISM (%d)", uint16(e))
	case EMEMQDSP6:
		return fmt.Sprintf("EMQDSP6 (%d)", uint16(e))
	case EMEMR32C:
		return fmt.Sprintf("EMR32C (%d)", uint16(e))
	case EMEMRCE:
		return fmt.Sprintf("EMRCE (%d)", uint16(e))
	case EMEMRH32:
		return fmt.Sprintf("EMRH32 (%d)", uint16(e))
	case EMEMRISCV:
		return fmt.Sprintf("EMRISCV (%d)", uint16(e))
	case EMEMRL78:
		return fmt.Sprintf("EMRL78 (%d)", uint16(e))
	case EMEMRS08:
		return fmt.Sprintf("EMRS08 (%d)", uint16(e))
	case EMEMRX:
		return fmt.Sprintf("EMRX (%d)", uint16(e))
	case EMEMS370:
		return fmt.Sprintf("EMS370 (%d)", uint16(e))
	case EMEMS390:
		return fmt.Sprintf("EMS390 (%d)", uint16(e))
	case EMEMSCORE7:
		return fmt.Sprintf("EMSCORE7 (%d)", uint16(e))
	case EMEMSEC17:
		return fmt.Sprintf("EMSEC17 (%d)", uint16(e))
	case EMEMSEC33:
		return fmt.Sprintf("EMSEC33 (%d)", uint16(e))
	case EMEMSEP:
		return fmt.Sprintf("EMSEP (%d)", uint16(e))
	case EMEMSH:
		return fmt.Sprintf("EMSH (%d)", uint16(e))
	case EMEMSHARC:
		return fmt.Sprintf("EMSHARC (%d)", uint16(e))
	case EMEMSLE9X:
		return fmt.Sprintf("EMSLE9X (%d)", uint16(e))
	case EMEMSNP1K:
		return fmt.Sprintf("EMSNP1K (%d)", uint16(e))
	case EMEMSPARC:
		return fmt.Sprintf("EMSPARC (%d)", uint16(e))
	case EMEMSPARC32PLUS:
		return fmt.Sprintf("EMSPARC32PLUS (%d)", uint16(e))
	case EMEMSPARCV9:
		return fmt.Sprintf("EMSPARCV9 (%d)", uint16(e))
	case EMEMSPU:
		return fmt.Sprintf("EMSPU (%d)", uint16(e))
	case EMEMST100:
		return fmt.Sprintf("EMST100 (%d)", uint16(e))
	case EMEMST19:
		return fmt.Sprintf("EMST19 (%d)", uint16(e))
	case EMEMST200:
		return fmt.Sprintf("EMST200 (%d)", uint16(e))
	case EMEMST7:
		return fmt.Sprintf("EMST7 (%d)", uint16(e))
	case EMEMST9PLUS:
		return fmt.Sprintf("EMST9PLUS (%d)", uint16(e))
	case EMEMSTARCORE:
		return fmt.Sprintf("EMSTARCORE (%d)", uint16(e))
	case EMEMSTM8:
		return fmt.Sprintf("EMSTM8 (%d)", uint16(e))
	case EMEMSTXP7X:
		return fmt.Sprintf("EMSTXP7X (%d)", uint16(e))
	case EMEMSVX:
		return fmt.Sprintf("EMSVX (%d)", uint16(e))
	case EMEMTIARP32:
		return fmt.Sprintf("EMTIARP32 (%d)", uint16(e))
	case EMEMTIC2000:
		return fmt.Sprintf("EMTIC2000 (%d)", uint16(e))
	case EMEMTIC5500:
		return fmt.Sprintf("EMTIC5500 (%d)", uint16(e))
	case EMEMTIC6000:
		return fmt.Sprintf("EMTIC6000 (%d)", uint16(e))
	case EMEMTILE64:
		return fmt.Sprintf("EMTILE64 (%d)", uint16(e))
	case EMEMTILEGX:
		return fmt.Sprintf("EMTILEGX (%d)", uint16(e))
	case EMEMTILEPRO:
		return fmt.Sprintf("EMTILEPRO (%d)", uint16(e))
	case EMEMTINYJ:
		return fmt.Sprintf("EMTINYJ (%d)", uint16(e))
	case EMEMTIPRU:
		return fmt.Sprintf("EMTIPRU (%d)", uint16(e))
	case EMEMTMMGPP:
		return fmt.Sprintf("EMTMMGPP (%d)", uint16(e))
	case EMEMTPC:
		return fmt.Sprintf("EMTPC (%d)", uint16(e))
	case EMEMTRICORE:
		return fmt.Sprintf("EMTRICORE (%d)", uint16(e))
	case EMEMTRIMEDIA:
		return fmt.Sprintf("EMTRIMEDIA (%d)", uint16(e))
	case EMEMTSK3000:
		return fmt.Sprintf("EMTSK3000 (%d)", uint16(e))
	case EMEMUNICORE:
		return fmt.Sprintf("EMUNICORE (%d)", uint16(e))
	case EMEMV800:
		return fmt.Sprintf("EMV800 (%d)", uint16(e))
	case EMEMV850:
		return fmt.Sprintf("EMV850 (%d)", uint16(e))
	case EMEMVAX:
		return fmt.Sprintf("EMVAX (%d)", uint16(e))
	case EMEMVIDEOCORE:
		return fmt.Sprintf("EMVIDEOCORE (%d)", uint16(e))
	case EMEMVIDEOCORE3:
		return fmt.Sprintf("EMVIDEOCORE3 (%d)", uint16(e))
	case EMEMVIDEOCORE5:
		return fmt.Sprintf("EMVIDEOCORE5 (%d)", uint16(e))
	case EMEMVISIUM:
		return fmt.Sprintf("EMVISIUM (%d)", uint16(e))
	case EMEMVPP500:
		return fmt.Sprintf("EMVPP500 (%d)", uint16(e))
	case EMEMX8664:
		return fmt.Sprintf("EMX8664 (%d)", uint16(e))
	case EMEMXCORE:
		return fmt.Sprintf("EMXCORE (%d)", uint16(e))
	case EMEMXGATE:
		return fmt.Sprintf("EMXGATE (%d)", uint16(e))
	case EMEMXIMO16:
		return fmt.Sprintf("EMXIMO16 (%d)", uint16(e))
	case EMEMXTENSA:
		return fmt.Sprintf("EMXTENSA (%d)", uint16(e))
	case EMEMZ80:
		return fmt.Sprintf("EMZ80 (%d)", uint16(e))
	case EMEMZSP:
		return fmt.Sprintf("EMZSP (%d)", uint16(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint16(e))
	}
}

func (e EM) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type SHT uint32

const (
	SHTNULL                 SHT = 0
	SHTPROGBITS             SHT = 1
	SHTSYMTAB               SHT = 2
	SHTSTRTAB               SHT = 3
	SHTRELA                 SHT = 4
	SHTHASH                 SHT = 5
	SHTDYNAMIC              SHT = 6
	SHTNOTE                 SHT = 7
	SHTNOBITS               SHT = 8
	SHTREL                  SHT = 9
	SHTSHLIB                SHT = 10
	SHTDYNSYM               SHT = 11
	SHTUNKNOWN12            SHT = 12
	SHTUNKNOWN13            SHT = 13
	SHTINITARRAY            SHT = 14
	SHTFINIARRAY            SHT = 15
	SHTPREINITARRAY         SHT = 16
	SHTGROUP                SHT = 17
	SHTSYMTABSHNDX          SHT = 18
	SHTGNUINCREMENTALINPUTS SHT = 1879000832
	SHTGNUATTRIBUTES        SHT = 1879048181
	SHTGNUHASH              SHT = 1879048182
	SHTGNULIBLIST           SHT = 1879048183
	SHTCHECKSUM             SHT = 1879048184
	SHTSUNWMove             SHT = 1879048186
	SHTSUNWCOMDAT           SHT = 1879048187
	SHTSUNWSyminfo          SHT = 1879048188
	SHTGNUVerdef            SHT = 1879048189
	SHTGNUVerneed           SHT = 1879048190
	SHTGNUVersym            SHT = 1879048191
	SHTARMEXIDX             SHT = 1879048193
	SHTARMPREEMPTMAP        SHT = 1879048194
	SHTARMATTRIBUTES        SHT = 1879048195
	SHTARMDEBUGOVERLAY      SHT = 1879048196
	SHTARMOVERLAYSECTION    SHT = 1879048197
)

func (e SHT) String() string {
	switch e {
	case SHTARMATTRIBUTES:
		return fmt.Sprintf("ARMATTRIBUTES (%d)", uint32(e))
	case SHTARMDEBUGOVERLAY:
		return fmt.Sprintf("ARMDEBUGOVERLAY (%d)", uint32(e))
	case SHTARMEXIDX:
		return fmt.Sprintf("ARMEXIDX (%d)", uint32(e))
	case SHTARMOVERLAYSECTION:
		return fmt.Sprintf("ARMOVERLAYSECTION (%d)", uint32(e))
	case SHTARMPREEMPTMAP:
		return fmt.Sprintf("ARMPREEMPTMAP (%d)", uint32(e))
	case SHTCHECKSUM:
		return fmt.Sprintf("CHECKSUM (%d)", uint32(e))
	case SHTDYNAMIC:
		return fmt.Sprintf("DYNAMIC (%d)", uint32(e))
	case SHTDYNSYM:
		return fmt.Sprintf("DYNSYM (%d)", uint32(e))
	case SHTFINIARRAY:
		return fmt.Sprintf("FINIARRAY (%d)", uint32(e))
	case SHTGNUATTRIBUTES:
		return fmt.Sprintf("GNUATTRIBUTES (%d)", uint32(e))
	case SHTGNUHASH:
		return fmt.Sprintf("GNUHASH (%d)", uint32(e))
	case SHTGNUINCREMENTALINPUTS:
		return fmt.Sprintf("GNUINCREMENTALINPUTS (%d)", uint32(e))
	case SHTGNULIBLIST:
		return fmt.Sprintf("GNULIBLIST (%d)", uint32(e))
	case SHTGNUVerdef:
		return fmt.Sprintf("GNUVerdef (%d)", uint32(e))
	case SHTGNUVerneed:
		return fmt.Sprintf("GNUVerneed (%d)", uint32(e))
	case SHTGNUVersym:
		return fmt.Sprintf("GNUVersym (%d)", uint32(e))
	case SHTGROUP:
		return fmt.Sprintf("GROUP (%d)", uint32(e))
	case SHTHASH:
		return fmt.Sprintf("HASH (%d)", uint32(e))
	case SHTINITARRAY:
		return fmt.Sprintf("INITARRAY (%d)", uint32(e))
	case SHTNOBITS:
		return fmt.Sprintf("NOBITS (%d)", uint32(e))
	case SHTNOTE:
		return fmt.Sprintf("NOTE (%d)", uint32(e))
	case SHTNULL:
		return fmt.Sprintf("NULL (%d)", uint32(e))
	case SHTPREINITARRAY:
		return fmt.Sprintf("PREINITARRAY (%d)", uint32(e))
	case SHTPROGBITS:
		return fmt.Sprintf("PROGBITS (%d)", uint32(e))
	case SHTREL:
		return fmt.Sprintf("REL (%d)", uint32(e))
	case SHTRELA:
		return fmt.Sprintf("RELA (%d)", uint32(e))
	case SHTSHLIB:
		return fmt.Sprintf("SHLIB (%d)", uint32(e))
	case SHTSTRTAB:
		return fmt.Sprintf("STRTAB (%d)", uint32(e))
	case SHTSUNWCOMDAT:
		return fmt.Sprintf("SUNWCOMDAT (%d)", uint32(e))
	case SHTSUNWMove:
		return fmt.Sprintf("SUNWMove (%d)", uint32(e))
	case SHTSUNWSyminfo:
		return fmt.Sprintf("SUNWSyminfo (%d)", uint32(e))
	case SHTSYMTAB:
		return fmt.Sprintf("SYMTAB (%d)", uint32(e))
	case SHTSYMTABSHNDX:
		return fmt.Sprintf("SYMTABSHNDX (%d)", uint32(e))
	case SHTUNKNOWN12:
		return fmt.Sprintf("UNKNOWN12 (%d)", uint32(e))
	case SHTUNKNOWN13:
		return fmt.Sprintf("UNKNOWN13 (%d)", uint32(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint32(e))
	}
}

func (e SHT) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type STV uint8

const (
	STVDEFAULT_  STV = 0
	STVINTERNAL  STV = 1
	STVHIDDEN    STV = 2
	STVPROTECTED STV = 3
)

func (e STV) String() string {
	switch e {
	case STVDEFAULT_:
		return fmt.Sprintf("DEFAULT_ (%d)", uint8(e))
	case STVHIDDEN:
		return fmt.Sprintf("HIDDEN (%d)", uint8(e))
	case STVINTERNAL:
		return fmt.Sprintf("INTERNAL (%d)", uint8(e))
	case STVPROTECTED:
		return fmt.Sprintf("PROTECTED (%d)", uint8(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint8(e))
	}
}

func (e STV) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type VERDEF uint16

const (
	VERDEFNON     VERDEF = 0
	VERDEFCURRENT VERDEF = 1
	VERDEFNUM     VERDEF = 2
)

func (e VERDEF) String() string {
	switch e {
	case VERDEFCURRENT:
		return fmt.Sprintf("CURRENT (%d)", uint16(e))
	case VERDEFNON:
		return fmt.Sprintf("NON (%d)", uint16(e))
	case VERDEFNUM:
		return fmt.Sprintf("NUM (%d)", uint16(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint16(e))
	}
}

func (e VERDEF) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type STT uint32

const (
	STTSTTNOTYPE   STT = 0
	STTSTTOBJECT   STT = 1
	STTSTTFUNC     STT = 2
	STTSTTSECTION  STT = 3
	STTSTTFILE     STT = 4
	STTSTTCOMMON   STT = 5
	STTSTTTLS      STT = 6
	STTSTTNUM      STT = 7
	STTSTTLOOS     STT = 10
	STTSTTGNUIFUNC STT = 10
	STTSTTHIOS     STT = 12
	STTSTTLOPROC   STT = 13
	STTSTTHIPROC   STT = 15
)

func (e STT) String() string {
	switch e {
	case STTSTTCOMMON:
		return fmt.Sprintf("STTCOMMON (%d)", uint32(e))
	case STTSTTFILE:
		return fmt.Sprintf("STTFILE (%d)", uint32(e))
	case STTSTTFUNC:
		return fmt.Sprintf("STTFUNC (%d)", uint32(e))
	case STTSTTGNUIFUNC:
		return fmt.Sprintf("STTGNUIFUNC (%d)", uint32(e))
	case STTSTTHIOS:
		return fmt.Sprintf("STTHIOS (%d)", uint32(e))
	case STTSTTHIPROC:
		return fmt.Sprintf("STTHIPROC (%d)", uint32(e))
	case STTSTTLOPROC:
		return fmt.Sprintf("STTLOPROC (%d)", uint32(e))
	case STTSTTNOTYPE:
		return fmt.Sprintf("STTNOTYPE (%d)", uint32(e))
	case STTSTTNUM:
		return fmt.Sprintf("STTNUM (%d)", uint32(e))
	case STTSTTOBJECT:
		return fmt.Sprintf("STTOBJECT (%d)", uint32(e))
	case STTSTTSECTION:
		return fmt.Sprintf("STTSECTION (%d)", uint32(e))
	case STTSTTTLS:
		return fmt.Sprintf("STTTLS (%d)", uint32(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint32(e))
	}
}

func (e STT) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type ET uint16

const (
	ETNONE ET = 0
	ETREL  ET = 1
	ETEXEC ET = 2
	ETDYN  ET = 3
	ETCORE ET = 4
)

func (e ET) String() string {
	switch e {
	case ETCORE:
		return fmt.Sprintf("CORE (%d)", uint16(e))
	case ETDYN:
		return fmt.Sprintf("DYN (%d)", uint16(e))
	case ETEXEC:
		return fmt.Sprintf("EXEC (%d)", uint16(e))
	case ETNONE:
		return fmt.Sprintf("NONE (%d)", uint16(e))
	case ETREL:
		return fmt.Sprintf("REL (%d)", uint16(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint16(e))
	}
}

func (e ET) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type DT uint32

const (
	DTDTNULL                    DT = 0
	DTDTNEEDED                  DT = 1
	DTDTPLTRELSZ                DT = 2
	DTDTPLTGOT                  DT = 3
	DTDTHASH                    DT = 4
	DTDTSTRTAB                  DT = 5
	DTDTSYMTAB                  DT = 6
	DTDTRELA                    DT = 7
	DTDTRELASZ                  DT = 8
	DTDTRELAENT                 DT = 9
	DTDTSTRSZ                   DT = 10
	DTDTSYMENT                  DT = 11
	DTDTINIT                    DT = 12
	DTDTFINI                    DT = 13
	DTDTSONAME                  DT = 14
	DTDTRPATH                   DT = 15
	DTDTSYMBOLIC                DT = 16
	DTDTREL                     DT = 17
	DTDTRELSZ                   DT = 18
	DTDTRELENT                  DT = 19
	DTDTPLTREL                  DT = 20
	DTDTDEBUG                   DT = 21
	DTDTTEXTREL                 DT = 22
	DTDTJMPREL                  DT = 23
	DTDTBINDNOW                 DT = 24
	DTDTINITARRAY               DT = 25
	DTDTFINIARRAY               DT = 26
	DTDTINITARRAYSZ             DT = 27
	DTDTFINIARRAYSZ             DT = 28
	DTDTRUNPATH                 DT = 29
	DTDTFLAGS                   DT = 30
	DTDTPREINITARRAY            DT = 32
	DTDTPREINITARRAYSZ          DT = 33
	DTDTMAXPOSTAGS              DT = 34
	DTDTNUM                     DT = 35
	DTDTGNUPRELINKED            DT = 1879047669
	DTDTGNUCONFLICTSZ           DT = 1879047670
	DTDTGNULIBLISTSZ            DT = 1879047671
	DTDTCHECKSUM                DT = 1879047672
	DTDTPLTPADSZ                DT = 1879047673
	DTDTMOVEENT                 DT = 1879047674
	DTDTMOVESZ                  DT = 1879047675
	DTDTFEATURE1                DT = 1879047676
	DTDTPOSFLAG1                DT = 1879047677
	DTDTSYMINSZ                 DT = 1879047678
	DTDTSYMINENT                DT = 1879047679
	DTDTGNUHASH                 DT = 1879047925
	DTDTTLSDESCPLT              DT = 1879047926
	DTDTTLSDESCGOT              DT = 1879047927
	DTDTGNUCONFLICT             DT = 1879047928
	DTDTGNULIBLIST              DT = 1879047929
	DTDTCONFIG                  DT = 1879047930
	DTDTDEPAUDIT                DT = 1879047931
	DTDTAUDIT                   DT = 1879047932
	DTDTPLTPAD                  DT = 1879047933
	DTDTMOVETAB                 DT = 1879047934
	DTDTSYMINFO                 DT = 1879047935
	DTDTVERSYM                  DT = 1879048176
	DTDTRELACOUNT               DT = 1879048185
	DTDTRELCOUNT                DT = 1879048186
	DTDTFLAGS1                  DT = 1879048187
	DTDTVERDEF                  DT = 1879048188
	DTDTVERDEFNUM               DT = 1879048189
	DTDTVERNEED                 DT = 1879048190
	DTDTVERNEEDNUM              DT = 1879048191
	DTDTAUXILIARY               DT = 2147483645
	DTDTUSED                    DT = 2147483646
	DTDTFILTER                  DT = 2147483647
	DTDTDEPRECATEDSPARCREGISTER DT = 117440513
	DTDTSUNWAUXILIARY           DT = 1610612749
	DTDTSUNWRTLDINF             DT = 1610612750
	DTDTSUNWFILTER              DT = 1610612751
	DTDTSUNWCAP                 DT = 1610612752
	DTDTSUNWSYMTAB              DT = 1610612753
	DTDTSUNWSYMSZ               DT = 1610612754
	DTDTSUNWSORTENT             DT = 1610612755
	DTDTSUNWSYMSORT             DT = 1610612756
	DTDTSUNWSYMSORTSZ           DT = 1610612757
	DTDTSUNWTLSSORT             DT = 1610612758
	DTDTSUNWTLSSORTSZ           DT = 1610612759
	DTDTSUNWSTRPAD              DT = 1610612761
	DTDTSUNWLDMACH              DT = 1610612763
)

func (e DT) String() string {
	switch e {
	case DTDTAUDIT:
		return fmt.Sprintf("DTAUDIT (%d)", uint32(e))
	case DTDTAUXILIARY:
		return fmt.Sprintf("DTAUXILIARY (%d)", uint32(e))
	case DTDTBINDNOW:
		return fmt.Sprintf("DTBINDNOW (%d)", uint32(e))
	case DTDTCHECKSUM:
		return fmt.Sprintf("DTCHECKSUM (%d)", uint32(e))
	case DTDTCONFIG:
		return fmt.Sprintf("DTCONFIG (%d)", uint32(e))
	case DTDTDEBUG:
		return fmt.Sprintf("DTDEBUG (%d)", uint32(e))
	case DTDTDEPAUDIT:
		return fmt.Sprintf("DTDEPAUDIT (%d)", uint32(e))
	case DTDTDEPRECATEDSPARCREGISTER:
		return fmt.Sprintf("DTDEPRECATEDSPARCREGISTER (%d)", uint32(e))
	case DTDTFEATURE1:
		return fmt.Sprintf("DTFEATURE1 (%d)", uint32(e))
	case DTDTFILTER:
		return fmt.Sprintf("DTFILTER (%d)", uint32(e))
	case DTDTFINI:
		return fmt.Sprintf("DTFINI (%d)", uint32(e))
	case DTDTFINIARRAY:
		return fmt.Sprintf("DTFINIARRAY (%d)", uint32(e))
	case DTDTFINIARRAYSZ:
		return fmt.Sprintf("DTFINIARRAYSZ (%d)", uint32(e))
	case DTDTFLAGS:
		return fmt.Sprintf("DTFLAGS (%d)", uint32(e))
	case DTDTFLAGS1:
		return fmt.Sprintf("DTFLAGS1 (%d)", uint32(e))
	case DTDTGNUCONFLICT:
		return fmt.Sprintf("DTGNUCONFLICT (%d)", uint32(e))
	case DTDTGNUCONFLICTSZ:
		return fmt.Sprintf("DTGNUCONFLICTSZ (%d)", uint32(e))
	case DTDTGNUHASH:
		return fmt.Sprintf("DTGNUHASH (%d)", uint32(e))
	case DTDTGNULIBLIST:
		return fmt.Sprintf("DTGNULIBLIST (%d)", uint32(e))
	case DTDTGNULIBLISTSZ:
		return fmt.Sprintf("DTGNULIBLISTSZ (%d)", uint32(e))
	case DTDTGNUPRELINKED:
		return fmt.Sprintf("DTGNUPRELINKED (%d)", uint32(e))
	case DTDTHASH:
		return fmt.Sprintf("DTHASH (%d)", uint32(e))
	case DTDTINIT:
		return fmt.Sprintf("DTINIT (%d)", uint32(e))
	case DTDTINITARRAY:
		return fmt.Sprintf("DTINITARRAY (%d)", uint32(e))
	case DTDTINITARRAYSZ:
		return fmt.Sprintf("DTINITARRAYSZ (%d)", uint32(e))
	case DTDTJMPREL:
		return fmt.Sprintf("DTJMPREL (%d)", uint32(e))
	case DTDTMAXPOSTAGS:
		return fmt.Sprintf("DTMAXPOSTAGS (%d)", uint32(e))
	case DTDTMOVEENT:
		return fmt.Sprintf("DTMOVEENT (%d)", uint32(e))
	case DTDTMOVESZ:
		return fmt.Sprintf("DTMOVESZ (%d)", uint32(e))
	case DTDTMOVETAB:
		return fmt.Sprintf("DTMOVETAB (%d)", uint32(e))
	case DTDTNEEDED:
		return fmt.Sprintf("DTNEEDED (%d)", uint32(e))
	case DTDTNULL:
		return fmt.Sprintf("DTNULL (%d)", uint32(e))
	case DTDTNUM:
		return fmt.Sprintf("DTNUM (%d)", uint32(e))
	case DTDTPLTGOT:
		return fmt.Sprintf("DTPLTGOT (%d)", uint32(e))
	case DTDTPLTPAD:
		return fmt.Sprintf("DTPLTPAD (%d)", uint32(e))
	case DTDTPLTPADSZ:
		return fmt.Sprintf("DTPLTPADSZ (%d)", uint32(e))
	case DTDTPLTREL:
		return fmt.Sprintf("DTPLTREL (%d)", uint32(e))
	case DTDTPLTRELSZ:
		return fmt.Sprintf("DTPLTRELSZ (%d)", uint32(e))
	case DTDTPOSFLAG1:
		return fmt.Sprintf("DTPOSFLAG1 (%d)", uint32(e))
	case DTDTPREINITARRAY:
		return fmt.Sprintf("DTPREINITARRAY (%d)", uint32(e))
	case DTDTPREINITARRAYSZ:
		return fmt.Sprintf("DTPREINITARRAYSZ (%d)", uint32(e))
	case DTDTREL:
		return fmt.Sprintf("DTREL (%d)", uint32(e))
	case DTDTRELA:
		return fmt.Sprintf("DTRELA (%d)", uint32(e))
	case DTDTRELACOUNT:
		return fmt.Sprintf("DTRELACOUNT (%d)", uint32(e))
	case DTDTRELAENT:
		return fmt.Sprintf("DTRELAENT (%d)", uint32(e))
	case DTDTRELASZ:
		return fmt.Sprintf("DTRELASZ (%d)", uint32(e))
	case DTDTRELCOUNT:
		return fmt.Sprintf("DTRELCOUNT (%d)", uint32(e))
	case DTDTRELENT:
		return fmt.Sprintf("DTRELENT (%d)", uint32(e))
	case DTDTRELSZ:
		return fmt.Sprintf("DTRELSZ (%d)", uint32(e))
	case DTDTRPATH:
		return fmt.Sprintf("DTRPATH (%d)", uint32(e))
	case DTDTRUNPATH:
		return fmt.Sprintf("DTRUNPATH (%d)", uint32(e))
	case DTDTSONAME:
		return fmt.Sprintf("DTSONAME (%d)", uint32(e))
	case DTDTSTRSZ:
		return fmt.Sprintf("DTSTRSZ (%d)", uint32(e))
	case DTDTSTRTAB:
		return fmt.Sprintf("DTSTRTAB (%d)", uint32(e))
	case DTDTSUNWAUXILIARY:
		return fmt.Sprintf("DTSUNWAUXILIARY (%d)", uint32(e))
	case DTDTSUNWCAP:
		return fmt.Sprintf("DTSUNWCAP (%d)", uint32(e))
	case DTDTSUNWFILTER:
		return fmt.Sprintf("DTSUNWFILTER (%d)", uint32(e))
	case DTDTSUNWLDMACH:
		return fmt.Sprintf("DTSUNWLDMACH (%d)", uint32(e))
	case DTDTSUNWRTLDINF:
		return fmt.Sprintf("DTSUNWRTLDINF (%d)", uint32(e))
	case DTDTSUNWSORTENT:
		return fmt.Sprintf("DTSUNWSORTENT (%d)", uint32(e))
	case DTDTSUNWSTRPAD:
		return fmt.Sprintf("DTSUNWSTRPAD (%d)", uint32(e))
	case DTDTSUNWSYMSORT:
		return fmt.Sprintf("DTSUNWSYMSORT (%d)", uint32(e))
	case DTDTSUNWSYMSORTSZ:
		return fmt.Sprintf("DTSUNWSYMSORTSZ (%d)", uint32(e))
	case DTDTSUNWSYMSZ:
		return fmt.Sprintf("DTSUNWSYMSZ (%d)", uint32(e))
	case DTDTSUNWSYMTAB:
		return fmt.Sprintf("DTSUNWSYMTAB (%d)", uint32(e))
	case DTDTSUNWTLSSORT:
		return fmt.Sprintf("DTSUNWTLSSORT (%d)", uint32(e))
	case DTDTSUNWTLSSORTSZ:
		return fmt.Sprintf("DTSUNWTLSSORTSZ (%d)", uint32(e))
	case DTDTSYMBOLIC:
		return fmt.Sprintf("DTSYMBOLIC (%d)", uint32(e))
	case DTDTSYMENT:
		return fmt.Sprintf("DTSYMENT (%d)", uint32(e))
	case DTDTSYMINENT:
		return fmt.Sprintf("DTSYMINENT (%d)", uint32(e))
	case DTDTSYMINFO:
		return fmt.Sprintf("DTSYMINFO (%d)", uint32(e))
	case DTDTSYMINSZ:
		return fmt.Sprintf("DTSYMINSZ (%d)", uint32(e))
	case DTDTSYMTAB:
		return fmt.Sprintf("DTSYMTAB (%d)", uint32(e))
	case DTDTTEXTREL:
		return fmt.Sprintf("DTTEXTREL (%d)", uint32(e))
	case DTDTTLSDESCGOT:
		return fmt.Sprintf("DTTLSDESCGOT (%d)", uint32(e))
	case DTDTTLSDESCPLT:
		return fmt.Sprintf("DTTLSDESCPLT (%d)", uint32(e))
	case DTDTUSED:
		return fmt.Sprintf("DTUSED (%d)", uint32(e))
	case DTDTVERDEF:
		return fmt.Sprintf("DTVERDEF (%d)", uint32(e))
	case DTDTVERDEFNUM:
		return fmt.Sprintf("DTVERDEFNUM (%d)", uint32(e))
	case DTDTVERNEED:
		return fmt.Sprintf("DTVERNEED (%d)", uint32(e))
	case DTDTVERNEEDNUM:
		return fmt.Sprintf("DTVERNEEDNUM (%d)", uint32(e))
	case DTDTVERSYM:
		return fmt.Sprintf("DTVERSYM (%d)", uint32(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint32(e))
	}
}

func (e DT) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type SHN uint16

const (
	SHNUNDEF  SHN = 0
	SHNBEFORE SHN = 65280
	SHNAFTER  SHN = 65281
	SHNABS    SHN = 65521
	SHNCOMMON SHN = 65522
	SHNXINDEX SHN = 65535
)

func (e SHN) String() string {
	switch e {
	case SHNABS:
		return fmt.Sprintf("ABS (%d)", uint16(e))
	case SHNAFTER:
		return fmt.Sprintf("AFTER (%d)", uint16(e))
	case SHNBEFORE:
		return fmt.Sprintf("BEFORE (%d)", uint16(e))
	case SHNCOMMON:
		return fmt.Sprintf("COMMON (%d)", uint16(e))
	case SHNUNDEF:
		return fmt.Sprintf("UNDEF (%d)", uint16(e))
	case SHNXINDEX:
		return fmt.Sprintf("XINDEX (%d)", uint16(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint16(e))
	}
}

func (e SHN) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type VERNDX uint16

const (
	VERNDXLOCAL     VERNDX = 0
	VERNDXGLOBAL    VERNDX = 1
	VERNDXELIMINATE VERNDX = 65281
)

func (e VERNDX) String() string {
	switch e {
	case VERNDXELIMINATE:
		return fmt.Sprintf("ELIMINATE (%d)", uint16(e))
	case VERNDXGLOBAL:
		return fmt.Sprintf("GLOBAL (%d)", uint16(e))
	case VERNDXLOCAL:
		return fmt.Sprintf("LOCAL (%d)", uint16(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint16(e))
	}
}

func (e VERNDX) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type VERNEED uint16

const (
	VERNEEDNONE    VERNEED = 0
	VERNEEDCURRENT VERNEED = 1
	VERNEEDNUM     VERNEED = 2
)

func (e VERNEED) String() string {
	switch e {
	case VERNEEDCURRENT:
		return fmt.Sprintf("CURRENT (%d)", uint16(e))
	case VERNEEDNONE:
		return fmt.Sprintf("NONE (%d)", uint16(e))
	case VERNEEDNUM:
		return fmt.Sprintf("NUM (%d)", uint16(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint16(e))
	}
}

func (e VERNEED) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type STB uint32

const (
	STBSTBLOCAL     STB = 0
	STBSTBGLOBAL    STB = 1
	STBSTBWEAK      STB = 2
	STBSTBNUM       STB = 3
	STBSTBLOOS      STB = 10
	STBSTBGNUUNIQUE STB = 10
	STBSTBHIOS      STB = 12
	STBSTBLOPROC    STB = 13
	STBSTBHIPROC    STB = 15
)

func (e STB) String() string {
	switch e {
	case STBSTBGLOBAL:
		return fmt.Sprintf("STBGLOBAL (%d)", uint32(e))
	case STBSTBGNUUNIQUE:
		return fmt.Sprintf("STBGNUUNIQUE (%d)", uint32(e))
	case STBSTBHIOS:
		return fmt.Sprintf("STBHIOS (%d)", uint32(e))
	case STBSTBHIPROC:
		return fmt.Sprintf("STBHIPROC (%d)", uint32(e))
	case STBSTBLOCAL:
		return fmt.Sprintf("STBLOCAL (%d)", uint32(e))
	case STBSTBLOPROC:
		return fmt.Sprintf("STBLOPROC (%d)", uint32(e))
	case STBSTBNUM:
		return fmt.Sprintf("STBNUM (%d)", uint32(e))
	case STBSTBWEAK:
		return fmt.Sprintf("STBWEAK (%d)", uint32(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint32(e))
	}
}

func (e STB) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type ELFCOMPRESS uint8

const (
	ELFCOMPRESSZLIB ELFCOMPRESS = 1
)

func (e ELFCOMPRESS) String() string {
	switch e {
	case ELFCOMPRESSZLIB:
		return fmt.Sprintf("ZLIB (%d)", uint8(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint8(e))
	}
}

func (e ELFCOMPRESS) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type EICLASS uint8

const (
	EICLASSELFCLASSNONE EICLASS = 0
	EICLASSELFCLASS32   EICLASS = 1
	EICLASSELFCLASS64   EICLASS = 2
)

func (e EICLASS) String() string {
	switch e {
	case EICLASSELFCLASS32:
		return fmt.Sprintf("ELFCLASS32 (%d)", uint8(e))
	case EICLASSELFCLASS64:
		return fmt.Sprintf("ELFCLASS64 (%d)", uint8(e))
	case EICLASSELFCLASSNONE:
		return fmt.Sprintf("ELFCLASSNONE (%d)", uint8(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint8(e))
	}
}

func (e EICLASS) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type EIDATA uint8

const (
	EIDATAELFDATANONE EIDATA = 0
	EIDATAELFDATA2LSB EIDATA = 1
	EIDATAELFDATA2MSB EIDATA = 2
)

func (e EIDATA) String() string {
	switch e {
	case EIDATAELFDATA2LSB:
		return fmt.Sprintf("ELFDATA2LSB (%d)", uint8(e))
	case EIDATAELFDATA2MSB:
		return fmt.Sprintf("ELFDATA2MSB (%d)", uint8(e))
	case EIDATAELFDATANONE:
		return fmt.Sprintf("ELFDATANONE (%d)", uint8(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint8(e))
	}
}

func (e EIDATA) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type EIVERSION uint8

const (
	EIVERSIONNONE    EIVERSION = 0
	EIVERSIONCURRENT EIVERSION = 1
)

func (e EIVERSION) String() string {
	switch e {
	case EIVERSIONCURRENT:
		return fmt.Sprintf("CURRENT (%d)", uint8(e))
	case EIVERSIONNONE:
		return fmt.Sprintf("NONE (%d)", uint8(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint8(e))
	}
}

func (e EIVERSION) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type RT uint32

const (
	RTCONSISTENT RT = 0
	RTADD        RT = 1
	RTDELETE     RT = 2
)

func (e RT) String() string {
	switch e {
	case RTADD:
		return fmt.Sprintf("ADD (%d)", uint32(e))
	case RTCONSISTENT:
		return fmt.Sprintf("CONSISTENT (%d)", uint32(e))
	case RTDELETE:
		return fmt.Sprintf("DELETE (%d)", uint32(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint32(e))
	}
}

func (e RT) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type PT uint32

const (
	PTNULL        PT = 0
	PTLOAD        PT = 1
	PTDYNAMIC     PT = 2
	PTINTERP      PT = 3
	PTNOTE        PT = 4
	PTSHLIB       PT = 5
	PTPHDR        PT = 6
	PTTLS         PT = 7
	PTLOOS        PT = 1610612736
	PTHIOS        PT = 1879048191
	PTGNUEHFRAME  PT = 1879048192
	PTGNUSTACK    PT = 1879048193
	PTGNURELRO    PT = 1879048194
	PTGNUPROPERTY PT = 1879048195
	PTSUNWBSS     PT = 1879048186
	PTSUNWSTACK   PT = 1879048187
	PTARMARCHEXT  PT = 1879048192
	PTARMUNWIND   PT = 1879048193
)

func (e PT) String() string {
	switch e {
	case PTARMARCHEXT:
		return fmt.Sprintf("ARMARCHEXT (%d)", uint32(e))
	case PTARMUNWIND:
		return fmt.Sprintf("ARMUNWIND (%d)", uint32(e))
	case PTDYNAMIC:
		return fmt.Sprintf("DYNAMIC (%d)", uint32(e))
	case PTGNUPROPERTY:
		return fmt.Sprintf("GNUPROPERTY (%d)", uint32(e))
	case PTGNURELRO:
		return fmt.Sprintf("GNURELRO (%d)", uint32(e))
	case PTHIOS:
		return fmt.Sprintf("HIOS (%d)", uint32(e))
	case PTINTERP:
		return fmt.Sprintf("INTERP (%d)", uint32(e))
	case PTLOAD:
		return fmt.Sprintf("LOAD (%d)", uint32(e))
	case PTLOOS:
		return fmt.Sprintf("LOOS (%d)", uint32(e))
	case PTNOTE:
		return fmt.Sprintf("NOTE (%d)", uint32(e))
	case PTNULL:
		return fmt.Sprintf("NULL (%d)", uint32(e))
	case PTPHDR:
		return fmt.Sprintf("PHDR (%d)", uint32(e))
	case PTSHLIB:
		return fmt.Sprintf("SHLIB (%d)", uint32(e))
	case PTSUNWBSS:
		return fmt.Sprintf("SUNWBSS (%d)", uint32(e))
	case PTSUNWSTACK:
		return fmt.Sprintf("SUNWSTACK (%d)", uint32(e))
	case PTTLS:
		return fmt.Sprintf("TLS (%d)", uint32(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint32(e))
	}
}

func (e PT) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type SYMINFOBT uint16

const (
	SYMINFOBTSELF   SYMINFOBT = 65535
	SYMINFOBTPARENT SYMINFOBT = 65534
	SYMINFOBTNONE   SYMINFOBT = 65533
)

func (e SYMINFOBT) String() string {
	switch e {
	case SYMINFOBTNONE:
		return fmt.Sprintf("NONE (%d)", uint16(e))
	case SYMINFOBTPARENT:
		return fmt.Sprintf("PARENT (%d)", uint16(e))
	case SYMINFOBTSELF:
		return fmt.Sprintf("SELF (%d)", uint16(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint16(e))
	}
}

func (e SYMINFOBT) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type EIOSABI uint8

const (
	EIOSABISYSV          EIOSABI = 0
	EIOSABIHPUX          EIOSABI = 1
	EIOSABINetBSD        EIOSABI = 2
	EIOSABILinux         EIOSABI = 3
	EIOSABIGNUHurd       EIOSABI = 4
	EIOSABISolaris       EIOSABI = 6
	EIOSABIAIX           EIOSABI = 7
	EIOSABIIRIX          EIOSABI = 8
	EIOSABIFreeBSD       EIOSABI = 9
	EIOSABITru64         EIOSABI = 10
	EIOSABINovellModesto EIOSABI = 11
	EIOSABIOpenBSD       EIOSABI = 12
	EIOSABIOpenVMS       EIOSABI = 13
	EIOSABINonStopKernel EIOSABI = 14
	EIOSABIAROS          EIOSABI = 15
	EIOSABIFenixOS       EIOSABI = 16
	EIOSABICloudABI      EIOSABI = 17
	EIOSABIOpenVOS       EIOSABI = 18
	EIOSABIARMEABI       EIOSABI = 64
	EIOSABISTANDALONE    EIOSABI = 255
)

func (e EIOSABI) String() string {
	switch e {
	case EIOSABIAIX:
		return fmt.Sprintf("AIX (%d)", uint8(e))
	case EIOSABIARMEABI:
		return fmt.Sprintf("ARMEABI (%d)", uint8(e))
	case EIOSABIAROS:
		return fmt.Sprintf("AROS (%d)", uint8(e))
	case EIOSABICloudABI:
		return fmt.Sprintf("CloudABI (%d)", uint8(e))
	case EIOSABIFenixOS:
		return fmt.Sprintf("FenixOS (%d)", uint8(e))
	case EIOSABIFreeBSD:
		return fmt.Sprintf("FreeBSD (%d)", uint8(e))
	case EIOSABIGNUHurd:
		return fmt.Sprintf("GNUHurd (%d)", uint8(e))
	case EIOSABIHPUX:
		return fmt.Sprintf("HPUX (%d)", uint8(e))
	case EIOSABIIRIX:
		return fmt.Sprintf("IRIX (%d)", uint8(e))
	case EIOSABILinux:
		return fmt.Sprintf("Linux (%d)", uint8(e))
	case EIOSABINetBSD:
		return fmt.Sprintf("NetBSD (%d)", uint8(e))
	case EIOSABINonStopKernel:
		return fmt.Sprintf("NonStopKernel (%d)", uint8(e))
	case EIOSABINovellModesto:
		return fmt.Sprintf("NovellModesto (%d)", uint8(e))
	case EIOSABIOpenBSD:
		return fmt.Sprintf("OpenBSD (%d)", uint8(e))
	case EIOSABIOpenVMS:
		return fmt.Sprintf("OpenVMS (%d)", uint8(e))
	case EIOSABIOpenVOS:
		return fmt.Sprintf("OpenVOS (%d)", uint8(e))
	case EIOSABISTANDALONE:
		return fmt.Sprintf("STANDALONE (%d)", uint8(e))
	case EIOSABISYSV:
		return fmt.Sprintf("SYSV (%d)", uint8(e))
	case EIOSABISolaris:
		return fmt.Sprintf("Solaris (%d)", uint8(e))
	case EIOSABITru64:
		return fmt.Sprintf("Tru64 (%d)", uint8(e))
	default:
		return fmt.Sprintf("unknown (%d)", uint8(e))
	}
}

func (e EIOSABI) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type SYMINFOFLG struct {
	DIRECT      bool
	RESERVED    bool
	COPY        bool
	LAZYLOAD    bool
	DIRECTBIND  bool
	NOEXTDIRECT bool
}

type ST struct {
	STBIND uint8
	STTYPE uint8
}

type SHF struct {
	WRITE           bool
	ALLOC           bool
	EXECINSTR       bool
	MERGE           bool
	STRINGS         bool
	INFOLINK        bool
	LINKORDER       bool
	OSNONCONFORMING bool
	GROUP           bool
	TLS             bool
	COMPRESSED      bool
	UNKNOWN         uint8
	MASKOS          uint8
	MASKPROC        uint8
}

type ELF32RINFO struct {
	TYPE_ uint8
	SYM   uint8
}

type ELF64RINFO struct {
	TYPE_ uint32
	SYM   uint32
}

type PF struct {
	X        bool
	W        bool
	R        bool
	MASKOS   uint8
	MASKPROC uint8
}

type Elf64Ehdr struct {
	EType      ET
	EMachine   EM
	EVersion   uint32
	EEntry     uint64
	EPhoff     uint64
	EShoff     uint64
	EFlags     uint32
	EEhsize    uint16
	EPhentsize uint16
	EPhnum     uint16
	EShentsize uint16
	EShnum     uint16
	EShstrndx  uint16
}

type Elf32Chdr struct {
	ChType      uint32
	ChSize      uint32
	ChAddralign uint32
}

type Elf64Chdr struct {
	ChType      uint32
	ChSize      uint32
	ChAddralign uint32
}

type Elf64Rel struct {
	ROffset uint64
	RInfo   ELF64RINFO
}

type Elf64Syminfo struct {
	SiBoundto uint16
	SiFlags   SYMINFOFLG
}

type EIDENT struct {
	EICLASS      EICLASS
	EIDATA       EIDATA
	EIVERSION    EIVERSION
	EIOSABI      EIOSABI
	EIABIVERSION uint8
}

type Elf32Ehdr struct {
	EType      ET
	EMachine   EM
	EVersion   uint32
	EEntry     uint32
	EPhoff     uint32
	EShoff     uint32
	EFlags     uint32
	EEhsize    uint16
	EPhentsize uint16
	EPhnum     uint16
	EShentsize uint16
	EShnum     uint16
	EShstrndx  uint16
}

type Elf32Rela struct {
	ROffset uint32
	RInfo   ELF32RINFO
	RAddend int32
}

type String struct {
	Value byte
}

type Elf32Shdr struct {
	ShName      uint32
	ShType      SHT
	ShFlags     SHF
	ShAddr      uint32
	ShOffset    uint32
	ShSize      uint32
	ShLink      uint32
	ShInfo      uint32
	ShAddralign uint32
	ShEntsize   uint32
}

type Elf64Phdr struct {
	PType   PT
	PFlags  PF
	POffset uint64
	PVaddr  uint64
	PPaddr  uint64
	PFilesz uint64
	PMemsz  uint64
	PAlign  uint64
}

type Elf32Sym struct {
	StName  uint32
	StValue uint32
	StSize  uint32
	StInfo  ST
	StOther STV
	StShndx uint16
}

type Elf32Syminfo struct {
	SiBoundto uint16
	SiFlags   SYMINFOFLG
}

type ELF struct {
	EIdent EIDENT
}

type Elf32Phdr struct {
	PType   PT
	POffset uint32
	PVaddr  uint32
	PPaddr  uint32
	PFilesz uint32
	PMemsz  uint32
	PFlags  PF
	PAlign  uint32
}

type Elf32Rel struct {
	ROffset uint32
	RInfo   ELF32RINFO
}

type Elf64Rela struct {
	ROffset uint64
	RInfo   ELF64RINFO
	RAddend int64
}

type Elf64Sym struct {
	StName  uint32
	StInfo  ST
	StOther STV
	StShndx uint16
	StValue uint64
	StSize  uint64
}

type Elf64Shdr struct {
	ShName      uint32
	ShType      SHT
	ShFlags     SHF
	ShAddr      uint64
	ShOffset    uint64
	ShSize      uint64
	ShLink      uint32
	ShInfo      uint32
	ShAddralign uint64
	ShEntsize   uint64
}

func ReadSYMINFOFLG(ctx *runtime.ReadContext, addr uintptr) (*SYMINFOFLG, runtime.Errors) {
	var errs runtime.Errors
	result := &SYMINFOFLG{}
	var buf [2]byte
	if _, err := ctx.ReadAt(buf[:], int64(addr)); err != nil {
		errs.Add("SYMINFOFLG", addr, err)
		return result, errs
	}
	raw := uint64(binary.LittleEndian.Uint16(buf[:]))
	result.DIRECT = (raw>>0)&1 != 0
	result.RESERVED = (raw>>1)&1 != 0
	result.COPY = (raw>>2)&1 != 0
	result.LAZYLOAD = (raw>>3)&1 != 0
	result.DIRECTBIND = (raw>>4)&1 != 0
	result.NOEXTDIRECT = (raw>>5)&1 != 0
	return result, errs
}

func ReadST(ctx *runtime.ReadContext, addr uintptr) (*ST, runtime.Errors) {
	var errs runtime.Errors
	result := &ST{}
	var buf [1]byte
	if _, err := ctx.ReadAt(buf[:], int64(addr)); err != nil {
		errs.Add("ST", addr, err)
		return result, errs
	}
	raw := uint64(buf[0])
	result.STBIND = uint8((raw >> 0) & 0xf)
	result.STTYPE = uint8((raw >> 4) & 0xf)
	return result, errs
}

func ReadSHF(ctx *runtime.ReadContext, addr uintptr) (*SHF, runtime.Errors) {
	var errs runtime.Errors
	result := &SHF{}
	var buf [4]byte
	if _, err := ctx.ReadAt(buf[:], int64(addr)); err != nil {
		errs.Add("SHF", addr, err)
		return result, errs
	}
	raw := uint64(binary.LittleEndian.Uint32(buf[:]))
	result.WRITE = (raw>>0)&1 != 0
	result.ALLOC = (raw>>1)&1 != 0
	result.EXECINSTR = (raw>>2)&1 != 0
	result.MERGE = (raw>>4)&1 != 0
	result.STRINGS = (raw>>5)&1 != 0
	result.INFOLINK = (raw>>6)&1 != 0
	result.LINKORDER = (raw>>7)&1 != 0
	result.OSNONCONFORMING = (raw>>8)&1 != 0
	result.GROUP = (raw>>9)&1 != 0
	result.TLS = (raw>>10)&1 != 0
	result.COMPRESSED = (raw>>11)&1 != 0
	result.UNKNOWN = uint8((raw >> 12) & 0xff)
	result.MASKOS = uint8((raw >> 20) & 0xff)
	result.MASKPROC = uint8((raw >> 28) & 0xf)
	return result, errs
}

func ReadELF32RINFO(ctx *runtime.ReadContext, addr uintptr) (*ELF32RINFO, runtime.Errors) {
	var errs runtime.Errors
	result := &ELF32RINFO{}
	var buf [4]byte
	if _, err := ctx.ReadAt(buf[:], int64(addr)); err != nil {
		errs.Add("ELF32RINFO", addr, err)
		return result, errs
	}
	raw := uint64(binary.LittleEndian.Uint32(buf[:]))
	result.TYPE_ = uint8((raw >> 0) & 0xff)
	result.SYM = uint8((raw >> 8) & 0xff)
	return result, errs
}

func ReadELF64RINFO(ctx *runtime.ReadContext, addr uintptr) (*ELF64RINFO, runtime.Errors) {
	var errs runtime.Errors
	result := &ELF64RINFO{}
	var buf [8]byte
	if _, err := ctx.ReadAt(buf[:], int64(addr)); err != nil {
		errs.Add("ELF64RINFO", addr, err)
		return result, errs
	}
	raw := binary.LittleEndian.Uint64(buf[:])
	result.TYPE_ = uint32((raw >> 0) & 0xffffffff)
	result.SYM = uint32((raw >> 32) & 0xffffffff)
	return result, errs
}

func ReadPF(ctx *runtime.ReadContext, addr uintptr) (*PF, runtime.Errors) {
	var errs runtime.Errors
	result := &PF{}
	var buf [4]byte
	if _, err := ctx.ReadAt(buf[:], int64(addr)); err != nil {
		errs.Add("PF", addr, err)
		return result, errs
	}
	raw := uint64(binary.LittleEndian.Uint32(buf[:]))
	result.X = (raw>>0)&1 != 0
	result.W = (raw>>1)&1 != 0
	result.R = (raw>>2)&1 != 0
	result.MASKOS = uint8((raw >> 20) & 0xf)
	result.MASKPROC = uint8((raw >> 24) & 0xf)
	return result, errs
}

func ReadElf64Ehdr(ctx *runtime.ReadContext, addr uintptr) (*Elf64Ehdr, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf64Ehdr{}
	var buf [8]byte

	// Field: EType (enum) at offset 0
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+0); err != nil {
		errs.Add("Elf64Ehdr.EType", uintptr(int64(addr)+0), err)
	} else {
		result.EType = ET(binary.LittleEndian.Uint16(buf[:2]))
	}

	// Field: EMachine (enum) at offset 2
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+2); err != nil {
		errs.Add("Elf64Ehdr.EMachine", uintptr(int64(addr)+2), err)
	} else {
		result.EMachine = EM(binary.LittleEndian.Uint16(buf[:2]))
	}

	// Field: EVersion at offset 4
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+4); err != nil {
		errs.Add("Elf64Ehdr.EVersion", uintptr(int64(addr)+4), err)
	} else {
		result.EVersion = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: EEntry at offset 8
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+8); err != nil {
		errs.Add("Elf64Ehdr.EEntry", uintptr(int64(addr)+8), err)
	} else {
		result.EEntry = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: EPhoff at offset 16
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+16); err != nil {
		errs.Add("Elf64Ehdr.EPhoff", uintptr(int64(addr)+16), err)
	} else {
		result.EPhoff = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: EShoff at offset 24
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+24); err != nil {
		errs.Add("Elf64Ehdr.EShoff", uintptr(int64(addr)+24), err)
	} else {
		result.EShoff = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: EFlags at offset 32
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+32); err != nil {
		errs.Add("Elf64Ehdr.EFlags", uintptr(int64(addr)+32), err)
	} else {
		result.EFlags = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: EEhsize at offset 36
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+36); err != nil {
		errs.Add("Elf64Ehdr.EEhsize", uintptr(int64(addr)+36), err)
	} else {
		result.EEhsize = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EPhentsize at offset 38
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+38); err != nil {
		errs.Add("Elf64Ehdr.EPhentsize", uintptr(int64(addr)+38), err)
	} else {
		result.EPhentsize = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EPhnum at offset 40
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+40); err != nil {
		errs.Add("Elf64Ehdr.EPhnum", uintptr(int64(addr)+40), err)
	} else {
		result.EPhnum = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EShentsize at offset 42
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+42); err != nil {
		errs.Add("Elf64Ehdr.EShentsize", uintptr(int64(addr)+42), err)
	} else {
		result.EShentsize = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EShnum at offset 44
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+44); err != nil {
		errs.Add("Elf64Ehdr.EShnum", uintptr(int64(addr)+44), err)
	} else {
		result.EShnum = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EShstrndx at offset 46
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+46); err != nil {
		errs.Add("Elf64Ehdr.EShstrndx", uintptr(int64(addr)+46), err)
	} else {
		result.EShstrndx = binary.LittleEndian.Uint16(buf[:2])
	}

	return result, errs
}

func ReadElf32Chdr(ctx *runtime.ReadContext, addr uintptr) (*Elf32Chdr, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf32Chdr{}
	var buf [4]byte

	// Field: ChType at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf32Chdr.ChType", uintptr(int64(addr)+0), err)
	} else {
		result.ChType = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ChSize at offset 4
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+4); err != nil {
		errs.Add("Elf32Chdr.ChSize", uintptr(int64(addr)+4), err)
	} else {
		result.ChSize = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ChAddralign at offset 8
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+8); err != nil {
		errs.Add("Elf32Chdr.ChAddralign", uintptr(int64(addr)+8), err)
	} else {
		result.ChAddralign = binary.LittleEndian.Uint32(buf[:4])
	}

	return result, errs
}

func ReadElf64Chdr(ctx *runtime.ReadContext, addr uintptr) (*Elf64Chdr, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf64Chdr{}
	var buf [4]byte

	// Field: ChType at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf64Chdr.ChType", uintptr(int64(addr)+0), err)
	} else {
		result.ChType = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ChSize at offset 4
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+4); err != nil {
		errs.Add("Elf64Chdr.ChSize", uintptr(int64(addr)+4), err)
	} else {
		result.ChSize = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ChAddralign at offset 8
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+8); err != nil {
		errs.Add("Elf64Chdr.ChAddralign", uintptr(int64(addr)+8), err)
	} else {
		result.ChAddralign = binary.LittleEndian.Uint32(buf[:4])
	}

	return result, errs
}

func ReadElf64Rel(ctx *runtime.ReadContext, addr uintptr) (*Elf64Rel, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf64Rel{}
	var buf [8]byte

	// Field: ROffset at offset 0
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+0); err != nil {
		errs.Add("Elf64Rel.ROffset", uintptr(int64(addr)+0), err)
	} else {
		result.ROffset = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: RInfo at offset 8
	{
		child, childErrs := ReadELF64RINFO(ctx, uintptr(int64(addr)+8))
		if child != nil {
			result.RInfo = *child
		}
		errs = append(errs, childErrs...)
	}

	return result, errs
}

func ReadElf64Syminfo(ctx *runtime.ReadContext, addr uintptr) (*Elf64Syminfo, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf64Syminfo{}
	var buf [2]byte

	// Field: SiBoundto at offset 0
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+0); err != nil {
		errs.Add("Elf64Syminfo.SiBoundto", uintptr(int64(addr)+0), err)
	} else {
		result.SiBoundto = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: SiFlags at offset 2
	{
		child, childErrs := ReadSYMINFOFLG(ctx, uintptr(int64(addr)+2))
		if child != nil {
			result.SiFlags = *child
		}
		errs = append(errs, childErrs...)
	}

	return result, errs
}

func ReadEIDENT(ctx *runtime.ReadContext, addr uintptr) (*EIDENT, runtime.Errors) {
	var errs runtime.Errors
	result := &EIDENT{}
	var buf [1]byte

	// Field: EICLASS (enum) at offset 4
	if _, err := ctx.ReadAt(buf[:1], int64(addr)+4); err != nil {
		errs.Add("EIDENT.EICLASS", uintptr(int64(addr)+4), err)
	} else {
		result.EICLASS = EICLASS(buf[0])
	}

	// Field: EIDATA (enum) at offset 5
	if _, err := ctx.ReadAt(buf[:1], int64(addr)+5); err != nil {
		errs.Add("EIDENT.EIDATA", uintptr(int64(addr)+5), err)
	} else {
		result.EIDATA = EIDATA(buf[0])
	}

	// Field: EIVERSION (enum) at offset 6
	if _, err := ctx.ReadAt(buf[:1], int64(addr)+6); err != nil {
		errs.Add("EIDENT.EIVERSION", uintptr(int64(addr)+6), err)
	} else {
		result.EIVERSION = EIVERSION(buf[0])
	}

	// Field: EIOSABI (enum) at offset 7
	if _, err := ctx.ReadAt(buf[:1], int64(addr)+7); err != nil {
		errs.Add("EIDENT.EIOSABI", uintptr(int64(addr)+7), err)
	} else {
		result.EIOSABI = EIOSABI(buf[0])
	}

	// Field: EIABIVERSION at offset 8
	if _, err := ctx.ReadAt(buf[:1], int64(addr)+8); err != nil {
		errs.Add("EIDENT.EIABIVERSION", uintptr(int64(addr)+8), err)
	} else {
		result.EIABIVERSION = buf[0]
	}

	return result, errs
}

func ReadElf32Ehdr(ctx *runtime.ReadContext, addr uintptr) (*Elf32Ehdr, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf32Ehdr{}
	var buf [4]byte

	// Field: EType (enum) at offset 0
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+0); err != nil {
		errs.Add("Elf32Ehdr.EType", uintptr(int64(addr)+0), err)
	} else {
		result.EType = ET(binary.LittleEndian.Uint16(buf[:2]))
	}

	// Field: EMachine (enum) at offset 2
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+2); err != nil {
		errs.Add("Elf32Ehdr.EMachine", uintptr(int64(addr)+2), err)
	} else {
		result.EMachine = EM(binary.LittleEndian.Uint16(buf[:2]))
	}

	// Field: EVersion at offset 4
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+4); err != nil {
		errs.Add("Elf32Ehdr.EVersion", uintptr(int64(addr)+4), err)
	} else {
		result.EVersion = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: EEntry at offset 8
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+8); err != nil {
		errs.Add("Elf32Ehdr.EEntry", uintptr(int64(addr)+8), err)
	} else {
		result.EEntry = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: EPhoff at offset 12
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+12); err != nil {
		errs.Add("Elf32Ehdr.EPhoff", uintptr(int64(addr)+12), err)
	} else {
		result.EPhoff = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: EShoff at offset 16
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+16); err != nil {
		errs.Add("Elf32Ehdr.EShoff", uintptr(int64(addr)+16), err)
	} else {
		result.EShoff = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: EFlags at offset 20
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+20); err != nil {
		errs.Add("Elf32Ehdr.EFlags", uintptr(int64(addr)+20), err)
	} else {
		result.EFlags = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: EEhsize at offset 24
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+24); err != nil {
		errs.Add("Elf32Ehdr.EEhsize", uintptr(int64(addr)+24), err)
	} else {
		result.EEhsize = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EPhentsize at offset 26
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+26); err != nil {
		errs.Add("Elf32Ehdr.EPhentsize", uintptr(int64(addr)+26), err)
	} else {
		result.EPhentsize = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EPhnum at offset 28
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+28); err != nil {
		errs.Add("Elf32Ehdr.EPhnum", uintptr(int64(addr)+28), err)
	} else {
		result.EPhnum = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EShentsize at offset 30
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+30); err != nil {
		errs.Add("Elf32Ehdr.EShentsize", uintptr(int64(addr)+30), err)
	} else {
		result.EShentsize = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EShnum at offset 32
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+32); err != nil {
		errs.Add("Elf32Ehdr.EShnum", uintptr(int64(addr)+32), err)
	} else {
		result.EShnum = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: EShstrndx at offset 34
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+34); err != nil {
		errs.Add("Elf32Ehdr.EShstrndx", uintptr(int64(addr)+34), err)
	} else {
		result.EShstrndx = binary.LittleEndian.Uint16(buf[:2])
	}

	return result, errs
}

func ReadElf32Rela(ctx *runtime.ReadContext, addr uintptr) (*Elf32Rela, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf32Rela{}
	var buf [4]byte

	// Field: ROffset at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf32Rela.ROffset", uintptr(int64(addr)+0), err)
	} else {
		result.ROffset = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: RInfo at offset 4
	{
		child, childErrs := ReadELF32RINFO(ctx, uintptr(int64(addr)+4))
		if child != nil {
			result.RInfo = *child
		}
		errs = append(errs, childErrs...)
	}

	// Field: RAddend at offset 8
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+8); err != nil {
		errs.Add("Elf32Rela.RAddend", uintptr(int64(addr)+8), err)
	} else {
		result.RAddend = int32(binary.LittleEndian.Uint32(buf[:4]))
	}

	return result, errs
}

func ReadString(ctx *runtime.ReadContext, addr uintptr) (*String, runtime.Errors) {
	var errs runtime.Errors
	result := &String{}
	var buf [1]byte

	// Field: Value at offset 0
	if _, err := ctx.ReadAt(buf[:1], int64(addr)+0); err != nil {
		errs.Add("String.Value", uintptr(int64(addr)+0), err)
	} else {
		result.Value = buf[0]
	}

	return result, errs
}

func ReadElf32Shdr(ctx *runtime.ReadContext, addr uintptr) (*Elf32Shdr, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf32Shdr{}
	var buf [4]byte

	// Field: ShName at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf32Shdr.ShName", uintptr(int64(addr)+0), err)
	} else {
		result.ShName = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShType (enum) at offset 4
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+4); err != nil {
		errs.Add("Elf32Shdr.ShType", uintptr(int64(addr)+4), err)
	} else {
		result.ShType = SHT(binary.LittleEndian.Uint32(buf[:4]))
	}

	// Field: ShFlags at offset 8
	{
		child, childErrs := ReadSHF(ctx, uintptr(int64(addr)+8))
		if child != nil {
			result.ShFlags = *child
		}
		errs = append(errs, childErrs...)
	}

	// Field: ShAddr at offset 12
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+12); err != nil {
		errs.Add("Elf32Shdr.ShAddr", uintptr(int64(addr)+12), err)
	} else {
		result.ShAddr = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShOffset at offset 16
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+16); err != nil {
		errs.Add("Elf32Shdr.ShOffset", uintptr(int64(addr)+16), err)
	} else {
		result.ShOffset = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShSize at offset 20
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+20); err != nil {
		errs.Add("Elf32Shdr.ShSize", uintptr(int64(addr)+20), err)
	} else {
		result.ShSize = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShLink at offset 24
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+24); err != nil {
		errs.Add("Elf32Shdr.ShLink", uintptr(int64(addr)+24), err)
	} else {
		result.ShLink = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShInfo at offset 28
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+28); err != nil {
		errs.Add("Elf32Shdr.ShInfo", uintptr(int64(addr)+28), err)
	} else {
		result.ShInfo = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShAddralign at offset 32
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+32); err != nil {
		errs.Add("Elf32Shdr.ShAddralign", uintptr(int64(addr)+32), err)
	} else {
		result.ShAddralign = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShEntsize at offset 36
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+36); err != nil {
		errs.Add("Elf32Shdr.ShEntsize", uintptr(int64(addr)+36), err)
	} else {
		result.ShEntsize = binary.LittleEndian.Uint32(buf[:4])
	}

	return result, errs
}

func ReadElf64Phdr(ctx *runtime.ReadContext, addr uintptr) (*Elf64Phdr, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf64Phdr{}
	var buf [8]byte

	// Field: PType (enum) at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf64Phdr.PType", uintptr(int64(addr)+0), err)
	} else {
		result.PType = PT(binary.LittleEndian.Uint32(buf[:4]))
	}

	// Field: PFlags at offset 4
	{
		child, childErrs := ReadPF(ctx, uintptr(int64(addr)+4))
		if child != nil {
			result.PFlags = *child
		}
		errs = append(errs, childErrs...)
	}

	// Field: POffset at offset 8
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+8); err != nil {
		errs.Add("Elf64Phdr.POffset", uintptr(int64(addr)+8), err)
	} else {
		result.POffset = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: PVaddr at offset 16
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+16); err != nil {
		errs.Add("Elf64Phdr.PVaddr", uintptr(int64(addr)+16), err)
	} else {
		result.PVaddr = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: PPaddr at offset 24
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+24); err != nil {
		errs.Add("Elf64Phdr.PPaddr", uintptr(int64(addr)+24), err)
	} else {
		result.PPaddr = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: PFilesz at offset 32
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+32); err != nil {
		errs.Add("Elf64Phdr.PFilesz", uintptr(int64(addr)+32), err)
	} else {
		result.PFilesz = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: PMemsz at offset 40
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+40); err != nil {
		errs.Add("Elf64Phdr.PMemsz", uintptr(int64(addr)+40), err)
	} else {
		result.PMemsz = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: PAlign at offset 48
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+48); err != nil {
		errs.Add("Elf64Phdr.PAlign", uintptr(int64(addr)+48), err)
	} else {
		result.PAlign = binary.LittleEndian.Uint64(buf[:8])
	}

	return result, errs
}

func ReadElf32Sym(ctx *runtime.ReadContext, addr uintptr) (*Elf32Sym, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf32Sym{}
	var buf [4]byte

	// Field: StName at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf32Sym.StName", uintptr(int64(addr)+0), err)
	} else {
		result.StName = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: StValue at offset 4
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+4); err != nil {
		errs.Add("Elf32Sym.StValue", uintptr(int64(addr)+4), err)
	} else {
		result.StValue = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: StSize at offset 8
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+8); err != nil {
		errs.Add("Elf32Sym.StSize", uintptr(int64(addr)+8), err)
	} else {
		result.StSize = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: StInfo at offset 12
	{
		child, childErrs := ReadST(ctx, uintptr(int64(addr)+12))
		if child != nil {
			result.StInfo = *child
		}
		errs = append(errs, childErrs...)
	}

	// Field: StOther (enum) at offset 13
	if _, err := ctx.ReadAt(buf[:1], int64(addr)+13); err != nil {
		errs.Add("Elf32Sym.StOther", uintptr(int64(addr)+13), err)
	} else {
		result.StOther = STV(buf[0])
	}

	// Field: StShndx at offset 14
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+14); err != nil {
		errs.Add("Elf32Sym.StShndx", uintptr(int64(addr)+14), err)
	} else {
		result.StShndx = binary.LittleEndian.Uint16(buf[:2])
	}

	return result, errs
}

func ReadElf32Syminfo(ctx *runtime.ReadContext, addr uintptr) (*Elf32Syminfo, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf32Syminfo{}
	var buf [2]byte

	// Field: SiBoundto at offset 0
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+0); err != nil {
		errs.Add("Elf32Syminfo.SiBoundto", uintptr(int64(addr)+0), err)
	} else {
		result.SiBoundto = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: SiFlags at offset 2
	{
		child, childErrs := ReadSYMINFOFLG(ctx, uintptr(int64(addr)+2))
		if child != nil {
			result.SiFlags = *child
		}
		errs = append(errs, childErrs...)
	}

	return result, errs
}

func ReadELF(ctx *runtime.ReadContext, addr uintptr) (*ELF, runtime.Errors) {
	var errs runtime.Errors
	result := &ELF{}

	// Field: EIdent at offset 0
	{
		child, childErrs := ReadEIDENT(ctx, uintptr(int64(addr)+0))
		if child != nil {
			result.EIdent = *child
		}
		errs = append(errs, childErrs...)
	}

	return result, errs
}

func ReadElf32Phdr(ctx *runtime.ReadContext, addr uintptr) (*Elf32Phdr, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf32Phdr{}
	var buf [4]byte

	// Field: PType (enum) at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf32Phdr.PType", uintptr(int64(addr)+0), err)
	} else {
		result.PType = PT(binary.LittleEndian.Uint32(buf[:4]))
	}

	// Field: POffset at offset 4
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+4); err != nil {
		errs.Add("Elf32Phdr.POffset", uintptr(int64(addr)+4), err)
	} else {
		result.POffset = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: PVaddr at offset 8
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+8); err != nil {
		errs.Add("Elf32Phdr.PVaddr", uintptr(int64(addr)+8), err)
	} else {
		result.PVaddr = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: PPaddr at offset 12
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+12); err != nil {
		errs.Add("Elf32Phdr.PPaddr", uintptr(int64(addr)+12), err)
	} else {
		result.PPaddr = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: PFilesz at offset 16
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+16); err != nil {
		errs.Add("Elf32Phdr.PFilesz", uintptr(int64(addr)+16), err)
	} else {
		result.PFilesz = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: PMemsz at offset 20
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+20); err != nil {
		errs.Add("Elf32Phdr.PMemsz", uintptr(int64(addr)+20), err)
	} else {
		result.PMemsz = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: PFlags at offset 24
	{
		child, childErrs := ReadPF(ctx, uintptr(int64(addr)+24))
		if child != nil {
			result.PFlags = *child
		}
		errs = append(errs, childErrs...)
	}

	// Field: PAlign at offset 28
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+28); err != nil {
		errs.Add("Elf32Phdr.PAlign", uintptr(int64(addr)+28), err)
	} else {
		result.PAlign = binary.LittleEndian.Uint32(buf[:4])
	}

	return result, errs
}

func ReadElf32Rel(ctx *runtime.ReadContext, addr uintptr) (*Elf32Rel, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf32Rel{}
	var buf [4]byte

	// Field: ROffset at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf32Rel.ROffset", uintptr(int64(addr)+0), err)
	} else {
		result.ROffset = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: RInfo at offset 4
	{
		child, childErrs := ReadELF32RINFO(ctx, uintptr(int64(addr)+4))
		if child != nil {
			result.RInfo = *child
		}
		errs = append(errs, childErrs...)
	}

	return result, errs
}

func ReadElf64Rela(ctx *runtime.ReadContext, addr uintptr) (*Elf64Rela, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf64Rela{}
	var buf [8]byte

	// Field: ROffset at offset 0
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+0); err != nil {
		errs.Add("Elf64Rela.ROffset", uintptr(int64(addr)+0), err)
	} else {
		result.ROffset = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: RInfo at offset 8
	{
		child, childErrs := ReadELF64RINFO(ctx, uintptr(int64(addr)+8))
		if child != nil {
			result.RInfo = *child
		}
		errs = append(errs, childErrs...)
	}

	// Field: RAddend at offset 16
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+16); err != nil {
		errs.Add("Elf64Rela.RAddend", uintptr(int64(addr)+16), err)
	} else {
		result.RAddend = int64(binary.LittleEndian.Uint64(buf[:8]))
	}

	return result, errs
}

func ReadElf64Sym(ctx *runtime.ReadContext, addr uintptr) (*Elf64Sym, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf64Sym{}
	var buf [8]byte

	// Field: StName at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf64Sym.StName", uintptr(int64(addr)+0), err)
	} else {
		result.StName = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: StInfo at offset 4
	{
		child, childErrs := ReadST(ctx, uintptr(int64(addr)+4))
		if child != nil {
			result.StInfo = *child
		}
		errs = append(errs, childErrs...)
	}

	// Field: StOther (enum) at offset 5
	if _, err := ctx.ReadAt(buf[:1], int64(addr)+5); err != nil {
		errs.Add("Elf64Sym.StOther", uintptr(int64(addr)+5), err)
	} else {
		result.StOther = STV(buf[0])
	}

	// Field: StShndx at offset 6
	if _, err := ctx.ReadAt(buf[:2], int64(addr)+6); err != nil {
		errs.Add("Elf64Sym.StShndx", uintptr(int64(addr)+6), err)
	} else {
		result.StShndx = binary.LittleEndian.Uint16(buf[:2])
	}

	// Field: StValue at offset 8
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+8); err != nil {
		errs.Add("Elf64Sym.StValue", uintptr(int64(addr)+8), err)
	} else {
		result.StValue = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: StSize at offset 16
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+16); err != nil {
		errs.Add("Elf64Sym.StSize", uintptr(int64(addr)+16), err)
	} else {
		result.StSize = binary.LittleEndian.Uint64(buf[:8])
	}

	return result, errs
}

func ReadElf64Shdr(ctx *runtime.ReadContext, addr uintptr) (*Elf64Shdr, runtime.Errors) {
	var errs runtime.Errors
	result := &Elf64Shdr{}
	var buf [8]byte

	// Field: ShName at offset 0
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+0); err != nil {
		errs.Add("Elf64Shdr.ShName", uintptr(int64(addr)+0), err)
	} else {
		result.ShName = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShType (enum) at offset 4
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+4); err != nil {
		errs.Add("Elf64Shdr.ShType", uintptr(int64(addr)+4), err)
	} else {
		result.ShType = SHT(binary.LittleEndian.Uint32(buf[:4]))
	}

	// Field: ShFlags at offset 8
	{
		child, childErrs := ReadSHF(ctx, uintptr(int64(addr)+8))
		if child != nil {
			result.ShFlags = *child
		}
		errs = append(errs, childErrs...)
	}

	// Field: ShAddr at offset 16
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+16); err != nil {
		errs.Add("Elf64Shdr.ShAddr", uintptr(int64(addr)+16), err)
	} else {
		result.ShAddr = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: ShOffset at offset 24
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+24); err != nil {
		errs.Add("Elf64Shdr.ShOffset", uintptr(int64(addr)+24), err)
	} else {
		result.ShOffset = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: ShSize at offset 32
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+32); err != nil {
		errs.Add("Elf64Shdr.ShSize", uintptr(int64(addr)+32), err)
	} else {
		result.ShSize = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: ShLink at offset 40
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+40); err != nil {
		errs.Add("Elf64Shdr.ShLink", uintptr(int64(addr)+40), err)
	} else {
		result.ShLink = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShInfo at offset 44
	if _, err := ctx.ReadAt(buf[:4], int64(addr)+44); err != nil {
		errs.Add("Elf64Shdr.ShInfo", uintptr(int64(addr)+44), err)
	} else {
		result.ShInfo = binary.LittleEndian.Uint32(buf[:4])
	}

	// Field: ShAddralign at offset 48
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+48); err != nil {
		errs.Add("Elf64Shdr.ShAddralign", uintptr(int64(addr)+48), err)
	} else {
		result.ShAddralign = binary.LittleEndian.Uint64(buf[:8])
	}

	// Field: ShEntsize at offset 56
	if _, err := ctx.ReadAt(buf[:8], int64(addr)+56); err != nil {
		errs.Add("Elf64Shdr.ShEntsize", uintptr(int64(addr)+56), err)
	} else {
		result.ShEntsize = binary.LittleEndian.Uint64(buf[:8])
	}

	return result, errs
}

// Ensure imports are used.
var (
	_ = binary.LittleEndian
	_ = json.Marshal
	_ = fmt.Sprintf
)
