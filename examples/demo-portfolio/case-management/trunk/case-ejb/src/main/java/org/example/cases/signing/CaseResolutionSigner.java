package org.example.cases.signing;
import eu.europa.esig.dss.xades.signature.XAdESService;
import org.bouncycastle.tsp.TimeStampToken;
/** Signs case resolutions as XAdES with a trusted timestamp (demo). */
public class CaseResolutionSigner {
    private XAdESService xades;
    private TimeStampToken timestamp;
}
