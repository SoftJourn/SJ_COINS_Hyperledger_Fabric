package com.softjourn.coins.gateway.service;

import com.softjourn.coins.gateway.config.ApplicationConstants;
import com.softjourn.coins.gateway.config.ApplicationProperties;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.hyperledger.fabric.gateway.Identities;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Wallet;
import org.hyperledger.fabric.gateway.X509Identity;
import org.hyperledger.fabric.gateway.impl.identity.GatewayUser;
import org.hyperledger.fabric.sdk.Enrollment;
import org.hyperledger.fabric.sdk.User;
import org.hyperledger.fabric.sdk.identity.X509Enrollment;
import org.hyperledger.fabric_ca.sdk.HFCAClient;
import org.hyperledger.fabric_ca.sdk.RegistrationRequest;
import org.springframework.stereotype.Service;

@Slf4j
@Service
@RequiredArgsConstructor
public class UserService {

  private final ApplicationProperties applicationProperties;
  private final HFCAClient caClient;
  private final Wallet wallet;
  private final AdminService adminService;

  public void enroll(String username) {
    try {
      Identity identity = wallet.get(username);

      if (identity != null) {
        log.info("An identity for the user '{}' already exists in the wallet", username);
        return;
      }

      Identity adminIdentity = adminService.getIdentity();
      if (adminIdentity == null) {
        log.warn(
            "An identity for the admin user '{}' does not exist in the wallet",
            ApplicationConstants.ADMIN_USERNAME);
        throw new IllegalStateException("Admin identity is not enrolled");
      }
      if (!(adminIdentity instanceof X509Identity)) {
        log.warn(
            "An identity for the admin user '{}' is not instance of X509 identity",
            ApplicationConstants.ADMIN_USERNAME);
        throw new IllegalStateException("Admin identity is not X509 identity");
      }

      X509Identity x509AdminIdentity = (X509Identity) adminIdentity;
      Enrollment adminEnrollment = new X509Enrollment(
          x509AdminIdentity.getPrivateKey(),
          Identities.toPemString(x509AdminIdentity.getCertificate()));

      RegistrationRequest regRequest =
          new RegistrationRequest(username, ApplicationConstants.USER_AFFILIATION);
      User registrar = new GatewayUser(
          ApplicationConstants.ADMIN_USERNAME, applicationProperties.getMspId(), adminEnrollment);

      String secret = caClient.register(regRequest, registrar);
      Enrollment enrollment = caClient.enroll(username, secret);

      identity = Identities.newX509Identity(applicationProperties.getMspId(), enrollment);
      wallet.put(username, identity);

      log.info(
          "Successfully registered and enrolled user '{}' and imported it into the wallet",
          username);
    } catch (Exception exception) {
      throw new RuntimeException(exception);
    }
  }
}
