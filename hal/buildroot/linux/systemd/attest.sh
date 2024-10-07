#!/bin/sh

function attest() {
    snpguest report attestation-report.bin request-data.txt --random

    snpguest fetch ca pem milan . --endorser vcek
    snpguest fetch vcek pem milan . attestation-report.bin

    # Verifies that ARK, ASK and VCEK are all properly signed
    snpguest verify certs .

    # Verifies the attestation-report trusted compute base matches vcek
    snpguest verify attestation . attestation-report.bin

    snpguest_report_measurement=$(snpguest display report attestation-report.bin | tr '\n' ' ' | sed "s|.*Measurement:\(.*\)Host Data.*|\1\n|g" | sed "s| ||g")
    # Remove any special characters and print the value
    snpguest_report_measurement=$(echo ${snpguest_report_measurement} | sed $'s/[^[:print:]\t]//g')
    echo -e "Measurement from SNP Attestation Report: ${snpguest_report_measurement}\n"
}

attest
