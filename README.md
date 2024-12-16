# lpac-routeros-wrapper
This is a wrapper for the [LPAC](https://github.com/estkme-group/lpac), to manage eSIM profiles in LTE interfaces for RouterOS.

# Usage
- Download and extract [LPAC](https://github.com/estkme-group/lpac/releases)
- Download and extract [lpac-routeros-wrapper](https://github.com/pccr10001/lpac-routeros-wrapper/releases)
- Copy `lpac-routeros-wrapper` to `lpac` folder
- Set environment variables for RouterOS connection.
  * Define environment variables with bash (`export`) or cmd (`set`)3
  * Define environment variables with `.env` file
- Run `lpac-routeros-wrapper chip info`

# Usage with other applications calling `lpac` command (ex. EasyLPAC)
- Download and extract [EasyLPAC](https://github.com/creamlike1024/EasyLPAC/releases)
- Download and extract [lpac-routeros-wrapper](https://github.com/pccr10001/lpac-routeros-wrapper/releases)
- Rename `lpac` in `EasyLPAC` directory to `lpac.orig`
- Copy `lpac-routeros-wrapper` to EasyLPAC
- Rename `lpac-fibocom-wrapper` to `lpac`
- Set environment variables for RouterOS connection.
    * Define environment variables with bash (`export`) or cmd (`set`)3
    * Define environment variables with `.env` file
- Launch EasyLPAC
- Select LTE interface in `Card Reader` menu
- Press `refresh` button

# Information
* Only MBIM mode is supported in RouterOS
  * This program uses `at-chat` command in RouterOS
  * For modules that supported in MBIM mode:
    * (*Tested*) Quectel EC20 / EC21 / EC25 / EC200
      * `AT+QCFG="usbnet",2` to switch into MBIM mode
    * (*Tested*) Fibocom L850-GL
      * See [xmm7360_usb](https://github.com/xmm7360/xmm7360-usb-modeswitch)
    * TBD

# Reference
* [lpac-fibocom-wrapper](https://github.com/prusa-dev/lpac-fibocom-wrapper)