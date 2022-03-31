package com.softjourn.common.helper;

import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import java.security.cert.CertificateException;
import java.security.cert.CertificateFactory;
import java.security.cert.X509Certificate;
import org.hyperledger.fabric.contract.Context;
import org.hyperledger.fabric.shim.ChaincodeException;

public class IdentityHelper {

  public String getCurrentUserId(final Context ctx) {
    String creatorCert = new String(ctx.getStub().getCreator(), StandardCharsets.UTF_8);

    int startIndex = creatorCert.indexOf("-----BEGIN CERTIFICATE-----");
    if (startIndex == -1) {
      startIndex = creatorCert.indexOf("-----BEGIN -----");
    }
    if (startIndex == -1) {
      throw new ChaincodeException("Invalid creator certificate");
    }

    String payload = creatorCert.substring(startIndex);
    X509Certificate cert = null;
    try {
      cert = (X509Certificate) CertificateFactory.getInstance("X.509")
          .generateCertificate(new ByteArrayInputStream(payload.getBytes()));

    } catch (CertificateException e) {
      throw new ChaincodeException(e.getMessage());
    }

    return cert.getSubjectDN().getName();
  }

  public String getUserAccount(final Context ctx, String accountType, String userId) {
    return ctx.getStub().createCompositeKey(accountType, userId).toString();
  }
}
