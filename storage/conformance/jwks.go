// This file was generaged by gen_jwks.go

package conformance

import "github.com/go-jose/go-jose/v4"

type keyPair struct {
	Public  *jose.JSONWebKey
	Private *jose.JSONWebKey
}

// keys are generated beforehand so we don't have to generate RSA keys for every test.
var jsonWebKeys = []keyPair{
	{
		Public: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "8145b5b9243c41459a8fdaa12acbd371",
			"alg": "RS256",
			"n": "34ls8E4onyEU_JKcxl8BMu2N6hK_D6aG2tOuCHJ_ka4rom8NmdJGdOQPC_fvKhcAxWeDktdAPislTT76Q4iMCC7DbM1aQhgRMaecKHBagc5ue2kSPM3oZPLqe6X-CxdxGTfXAvFIZM9JZTbQeJPcXFdn28iZ086xWPMdQKY5QTRKtoHQSN6EAQuuiuZsXrAC3lBZmE4tda6NoeYLb0UayGqiiFmtoIFJQ4NecI-EECT-mcjkPGWG0Ll5dCIUhGDl8sQSUrmBuaTDpPEzLGo-UtM3ay7AN0gOVN0mLIk2oyroXcVOA626LYNLVU0mz9PDpdkhWBeUfLL6i4HjUS3RaQ",
			"e": "AQAB"
		}`),
		Private: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "f547defc90b34ec08caeb8b294591216",
			"alg": "RS256",
			"n": "34ls8E4onyEU_JKcxl8BMu2N6hK_D6aG2tOuCHJ_ka4rom8NmdJGdOQPC_fvKhcAxWeDktdAPislTT76Q4iMCC7DbM1aQhgRMaecKHBagc5ue2kSPM3oZPLqe6X-CxdxGTfXAvFIZM9JZTbQeJPcXFdn28iZ086xWPMdQKY5QTRKtoHQSN6EAQuuiuZsXrAC3lBZmE4tda6NoeYLb0UayGqiiFmtoIFJQ4NecI-EECT-mcjkPGWG0Ll5dCIUhGDl8sQSUrmBuaTDpPEzLGo-UtM3ay7AN0gOVN0mLIk2oyroXcVOA626LYNLVU0mz9PDpdkhWBeUfLL6i4HjUS3RaQ",
			"e": "AQAB",
			"d": "3rABHsQ-I4jJZ3SHSfeLMjkFj5JtVCIJZiZK0Y9_Fpn0TjVjz0Fzfy9S7hFo6P1Rf1bH9JkLHuPMnU-H8Y8uMVikxtcse3uOZXEcWAzVnUsRNVBPItPeF_MHNXb_xfzsZrsCL6Q_Am6eJ36b4AMtG7DXflQxKphWhM5s7eKqVxDrkhaDPnALLRFjCvUZ_myQQ3Upn7gMgAbvfIY1fn9rXW_4CfxbxhcPJW5IOcu6bPvpQlfuFkXjF-gGCiNf5kv6Db0lpDOKX5l5T-KFGQ0dIOdasm8vL2GxCKZf55rKRCt0a28fwwH2p94ja-1qtPTc34V8F26LyVRgQgD3e-0aoQ",
			"p": "_WoAr3sgL5yfaqBL38yqx4hqSPZGdR6xTS64rhgZaVg14_W6xYmlPI7PmVBRW45Fk4tXhXjv9oMZH9HGrH2v4yqXLEq0gJr4VAPvRaN6p_kb_eCfLHCbNCYBAPNVUdFpOTvOmh7m0zYPrku7DZDnZQEN_A9hYcufjy0em-lV6Tc",
			"q": "4dFfwyYQmns1xwVEPABxpazk6nAluS-7yYSAc9A8D25nqm0mNWdPJvmpJS02xSDjIGfe0FtMr1XlPm3XHdUlIu2Z9Ex-J-kcs3lfs2UKmleQqJRXK4MahAEIV3vp0zG47hAJyzE3Oh4sVLFr3ZK9_-SenolCFv5eIikWa3Xg6l8"
		}`),
	},

	{
		Public: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "3a9365e41b114ec1b9288b214196e158",
			"alg": "RS256",
			"n": "t3TrxLN5_z-x5X9kebkoPnoYnGAPqAXOVCGBTxcAqev_P8t6SyyeeITDiePhCctYp5dO-WHRkB7_BkUeHZOgoyCBarDkDifQSG7MCtlYDm0yiSij_0vqzJQx-6zlXb5ypwO0P1sAXrO_nO87u69w5yaKf0yEJMpSjU8BDKQ__nskZP2QJJsYwOeAI9aAM2oP8r7Im8KzLy9-mnFSqypxBnL24hFNzKOS_GyHs0tPLjVY7JNDtDOkwPQIQFzsdZSY88n6uYvV-MGu3O-Y3-xLwUqMlJOXFskhmp1AOUnb4JgQ9wEaZ7088PY3Ak0eZkrg2FQ3XRHSWhUCOb2xL5iTvw",
			"e": "AQAB"
		}`),
		Private: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "c79418aaf8ee439bb2b0e28672d71584",
			"alg": "RS256",
			"n": "t3TrxLN5_z-x5X9kebkoPnoYnGAPqAXOVCGBTxcAqev_P8t6SyyeeITDiePhCctYp5dO-WHRkB7_BkUeHZOgoyCBarDkDifQSG7MCtlYDm0yiSij_0vqzJQx-6zlXb5ypwO0P1sAXrO_nO87u69w5yaKf0yEJMpSjU8BDKQ__nskZP2QJJsYwOeAI9aAM2oP8r7Im8KzLy9-mnFSqypxBnL24hFNzKOS_GyHs0tPLjVY7JNDtDOkwPQIQFzsdZSY88n6uYvV-MGu3O-Y3-xLwUqMlJOXFskhmp1AOUnb4JgQ9wEaZ7088PY3Ak0eZkrg2FQ3XRHSWhUCOb2xL5iTvw",
			"e": "AQAB",
			"d": "T7-y0dIXQV8l7RbAza0wkmAvHKMhiy_i7m2WMZRVRIiDb-77HXyq8sb73ZBC_if4RPogaYYdPCJNSCN5oO_Qz7jMqV119bVW9HW9myW6AqNzaW5SRCNzUTVGuRoCpwqn-nRAwZ3EfmZy8DyK4d61HLaDVC0l8HxHAIiMcztfWjbfD2LjwWF2hF5VRG2-haDfT6Kwtz0zEXblvYxyPqVyKOFtuWDlzX8iP8_ryWaChpR-jTmwtm7663wcu4M9teMkdgubCIqkz0LLtd-97ZUM2ti70WO7AEqE6p1evnjfYt4HZpQlsn0psrgGLvX2oCIvmPQMfTjzmtsEC51F5CU-yQ",
			"p": "4xi5OdCP9n1ivD3CuMhcaoMrwkC1yVdYnJwaNXjIyuSUT0i_QmuRpViydpZsfiYEoNNczL_PwxlDNdl2ccbelBuoEDbrvAfz0G0-YVYuLJoEKQs_OjenIn_6AZlmn7zSQ0LjoZ1tTjOaKuueB2b8RVtF2pbZ_o1ApyWd3q6QjyU",
			"q": "zs5SF-jdzP9xThPTEmAa2yh6SI48KuwVwWXGjOQZThXVEfwo-iZNevPjg3b6gwY9fKi71-J75c1ng0QrgdDuRIackHFpSLaWgcIpN31-uyZl5X-uxpBZON1HeiYT8J2JhgbA9ZJ0_SUq3j4YSrFEGKSpBi741mqwS9CZ6NSN5BM"
		}`),
	},

	{
		Public: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "4c267fc23c7b44d6973a1722b7201849",
			"alg": "RS256",
			"n": "yeZexEF1gOXd71iz9jRQR3EhgM2-o3mVO4O1fJYYQTh5APfrrbMhOGLvgK06vytREiY9_1awL7YfEnZzQynq9WTZpkwlAhYujHYf1RbGPeoXJS2cXKThfIhbeITEyhfepqzwU_f-RhvaLS3bydDi7F74oTO9njtLkGV2qNHH3B2uTFBy2G8VmDeHNQrUa868LQ9omrmWFkLnoZOoVPiLZD-5aZXOKJ0In5sg9B1EX1oaF-xejCTBX_8EJvvvKXH-GUZnHc3g3Rf3k4iXCJi8VMyjA8we3fgP8jp2P3Ofv6VOKG3vh8j5lI3ys_rctc2fu6CaNWNNZs9wbjpDVPuc0w",
			"e": "AQAB"
		}`),
		Private: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "eec6ee158cb34d699be4baff419da383",
			"alg": "RS256",
			"n": "yeZexEF1gOXd71iz9jRQR3EhgM2-o3mVO4O1fJYYQTh5APfrrbMhOGLvgK06vytREiY9_1awL7YfEnZzQynq9WTZpkwlAhYujHYf1RbGPeoXJS2cXKThfIhbeITEyhfepqzwU_f-RhvaLS3bydDi7F74oTO9njtLkGV2qNHH3B2uTFBy2G8VmDeHNQrUa868LQ9omrmWFkLnoZOoVPiLZD-5aZXOKJ0In5sg9B1EX1oaF-xejCTBX_8EJvvvKXH-GUZnHc3g3Rf3k4iXCJi8VMyjA8we3fgP8jp2P3Ofv6VOKG3vh8j5lI3ys_rctc2fu6CaNWNNZs9wbjpDVPuc0w",
			"e": "AQAB",
			"d": "IOFck5eZfElzMFSA0lrIrCnXa_OV1WeqjwuvFcAX6R86TZcSkbI3echa-ti7VYDHbi4-MIQ8oziErOwPb2O3OQmYjIWgDUvxfryKCJjx5glmhY59BXVwp2hJhUISDlt-ziQh63ratS46BNuQDLjxC8-XrCESA1_iuXxcq7emVclRKN2DpGehf2bZyjcZy-OEwvL1jLsvoY2jmY_2JOT4nFLqoelg5vENj69p8IR9Bpdzp0urngLZJ4-HqFGyfx3tEo4ZUF1M5xnoycBc5LMZjmElK66rjBRWPq9UwZwfqaeQh6wEA9siYw1V9yrNRUkq3Q6BErbXNDKBV36bRIiaIQ",
			"p": "_YirCr3Sfs9FkEFFMNsTZ2Wv8e5napONPtg1WUYOxG36k65EkPtlmZLWmiwmBk6592oND_S5WvbW4BbX5lRbEvNiRy9coVPst6lOOnLe69GJoI_GxoRyu_94qIS-VNPSQkyw4gfA1M-lMdfKpaTMv7fvVolvmDs5xN_fmXpl06M",
			"q": "y90gdyUcYzDX1u3-fCINzXbDcr80QEO3bjuG8p7feaYY2MP51t6j6MisNsQqcGKY7xFhpc-z8_cEIg1HJ3FSly-yejPj8RGavPX6NVGVHDNGwxxnm_i3kf-4MuDxwRSSHMlgVNAXuoH-3iicz-bNTVYM-5bYucZMvZHC6Ur2JRE"
		}`),
	},

	{
		Public: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "e10385c5384046f395fc6d9027db2f35",
			"alg": "RS256",
			"n": "299cgJgPiu9CK8hGgQw3j8e-Y_u4-Tm6WXKOFHdjCUPV5EAWMOa34cQNt75KN8pxlIcnujnU6TpH4OPRCw1gA44rrk_uczIEULsTnt6UFuMtUY2r-2UW2BWg5rEHyLcNX_QCA80T9DVSxsWeN8S23YcVk9fVputIRU7ee7auOx3b6K3pkoQJBVUk-_ndaqwlX-JU2CQG52CH91CrDzN0WGUPrhMZOdL7ybv94l5ztBrnjaQupkt0FxTA1_m_tXTvxIgzzegaqXrJ1mJM-z2TxPUJUc_04JaGilPUkxU780jk_03d46Op-pdElgbZ52C9JT9b8nRnA-vHq4e2whY8Yw",
			"e": "AQAB"
		}`),
		Private: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "8165cc507cd1492394be64575dfa8261",
			"alg": "RS256",
			"n": "299cgJgPiu9CK8hGgQw3j8e-Y_u4-Tm6WXKOFHdjCUPV5EAWMOa34cQNt75KN8pxlIcnujnU6TpH4OPRCw1gA44rrk_uczIEULsTnt6UFuMtUY2r-2UW2BWg5rEHyLcNX_QCA80T9DVSxsWeN8S23YcVk9fVputIRU7ee7auOx3b6K3pkoQJBVUk-_ndaqwlX-JU2CQG52CH91CrDzN0WGUPrhMZOdL7ybv94l5ztBrnjaQupkt0FxTA1_m_tXTvxIgzzegaqXrJ1mJM-z2TxPUJUc_04JaGilPUkxU780jk_03d46Op-pdElgbZ52C9JT9b8nRnA-vHq4e2whY8Yw",
			"e": "AQAB",
			"d": "xh587o6WKr2uZV8gUHXettroLpWKtl-TD7hOWBi_j4ClgfdRR50NggwzxCZeH-l18LzcSkyEEefnDriZC5lws6NurrHtjbU6-Dep1VSAIiNwGXVLy8nqDKlog5ZvCigPkC-BhUVMPpexz9QP3faORAzNn5szNCX7yB_qD5WrZy20AUEoWtGPgxGW6xf5Lgu6zg2uQEEB1Z0hKjHV9seIiuQooMrSzpS1D7BLSTHOvM2Y2lXvQQokc3uQXnyT_soHPjHl00bcuJLJaRCmyHRTol7uh9MNe67eMy7pHYmmlwOvTDfW6meKCgoEXd1wKIrS9VRY7WP36ZRpJH6qv8vceQ",
			"p": "7sWEsknUaSlAJ-bGhsuFr_j15zupV9O-DLnLobASm4Z7Ylt1HhtPN1NCVzYFTCtltPBE_CXGaAPqw3wiERK3tgYSLV8yk57sU1H28Zsq65A1B-vdlO69-F_6djiGegYKTOO4CXt0VYB4hJ6Trwx_BNJmrAD_Ykjqsp5sR0gOrqU",
			"q": "67y_hzbi81IH2DxmHTQOfHgcLYe-TnrEQLGLQtfx8J0J_REf_fLBDL-pt_jy6WIvTAb-LgUIcieiXfhni1nPUw0f_I1SDNv02EYvP0vkfyQdJBR6sLi4jv0mpqyQxvGif9B4eM9Qjngm2Jclj3-el-RkMZOUyf3zGTNGLI3MmGc"
		}`),
	},

	{
		Public: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "941861b40500430da0d09ec213e00832",
			"alg": "RS256",
			"n": "ub3SiNK-uIvSrUTyIPm1cITzuqPX_CIa6nZTDTP1tJ6PP_KufYz2eGLj9jppWLo_J7XQfKfIAKvET8Mq4HEcLQpNRN90KNyGML17JJtSgYJeLuB38BnalVUxpnycPKeGgoNJMu6t8tKYOtOfxtqTA6x8MnqMeify1cvEc5Tr4QmKjcLLHKcr1yMR7kG48i586bLdchtIBYeB298WXbQaKrgsEjZA0E1exfMnYHyvN12lMBxwhOJtcFu3mngZ7vTh179UKsP3yD8IdO5ITe_RIOmnUKuynW3PdkRUzCK5gS-xuqueGqEzJVIKBv0Hfom3eyDW5DjxpIZxlqkGhGyeNw",
			"e": "AQAB"
		}`),
		Private: mustLoadJWK(`{
			"use": "sig",
			"kty": "RSA",
			"kid": "c4c09817da9a42ae8d850aaba7b7cd82",
			"alg": "RS256",
			"n": "ub3SiNK-uIvSrUTyIPm1cITzuqPX_CIa6nZTDTP1tJ6PP_KufYz2eGLj9jppWLo_J7XQfKfIAKvET8Mq4HEcLQpNRN90KNyGML17JJtSgYJeLuB38BnalVUxpnycPKeGgoNJMu6t8tKYOtOfxtqTA6x8MnqMeify1cvEc5Tr4QmKjcLLHKcr1yMR7kG48i586bLdchtIBYeB298WXbQaKrgsEjZA0E1exfMnYHyvN12lMBxwhOJtcFu3mngZ7vTh179UKsP3yD8IdO5ITe_RIOmnUKuynW3PdkRUzCK5gS-xuqueGqEzJVIKBv0Hfom3eyDW5DjxpIZxlqkGhGyeNw",
			"e": "AQAB",
			"d": "29bQWSEWm1bjBDGWY3EqTwMNdtp1yPaU5O0nX3kgV6dT5VxXKkKtdc-WANkh1uKZ3WZUXTY4gpLKx504Im2965FF4z6XPcXFDes21R0BikfDMbh8PLJdBGLRYTwbr66YheDdwmq9d6nKg9X2RmZtmuuMFDL4EZ02zdVfr22TwcSCghC2gnV6CpHHeEatJBWbK1yE6cHqCeY9UTc_QnXmbZ0TYsQi4qCV1HqTJKZDtkzqZMPvMB5EP_my_SCxcfcIzt6qqujmuXCFiS658Up-Z4W5s0RINLoPmePG8zJVFBmWrQ8xiykCeL8z9XSvXoEo6ZJJC-KSjI6s-KsCfQqZ",
			"p": "8LzUJM2YgP7zG618rrFTav3gB2t1yMwFJy9d3J-pOkVFUq-4-74qEZz6H2RTUw7Ae5XEYdVIbRRQInpo0qO2MfLW8vtRexUNFFt1pBiVykq-KdkWcwPETyRD-huEEqswBhg33lFTUrY7BXRukbfNmVY7YfdagIJ5LZU0I-nGMqs",
			"q": "xYRoIFTTiXitKBFo0vvHAadqVV8gJq8bCxJZ4lFMpADlU-S8Me7aPmkhPmCDaw-ii940S46bTp9ueh6EJCttmG3cJm8r4YzjK-H1dnqeF_3dpq2pimVFlFILBKWojUHHWC4n0d1IVwdf8-xnDSiUzl9roFZV5IPy4mW1HMTZ4qU"
		}`),
	},
}
