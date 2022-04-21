package com.softjourn.coins.gateway.service;

import com.softjourn.coins.gateway.config.ApplicationProperties;
import io.grpc.netty.shaded.io.netty.handler.ssl.util.LazyX509Certificate;
import java.io.IOException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Wallet;
import org.hyperledger.fabric.gateway.impl.identity.X509IdentityImpl;
import org.hyperledger.fabric.sdk.Enrollment;
import org.hyperledger.fabric_ca.sdk.HFCAClient;
import org.hyperledger.fabric_ca.sdk.exception.EnrollmentException;
import org.hyperledger.fabric_ca.sdk.exception.InvalidArgumentException;
import org.springframework.stereotype.Service;

@Slf4j
@Service
@RequiredArgsConstructor
public class AdminService {

  private static final String ADMIN_NAME = "admin";

  private final ApplicationProperties applicationProperties;
  private final HFCAClient caClient;
  private final Wallet wallet;

  public void enroll() throws EnrollmentException {
    try {
      Identity identity = getIdentity();

      if (identity != null) {
        log.info("An identity for the admin user '{}' already exists in the wallet", ADMIN_NAME);
        return;
      }

      Enrollment enrollment = caClient.enroll(
          applicationProperties.getAdminUsername(), applicationProperties.getAdminPassword());

      identity = new X509IdentityImpl(
          applicationProperties.getMspId(),
          new LazyX509Certificate(enrollment.getCert().getBytes()),
          enrollment.getKey()
      );
      wallet.put(ADMIN_NAME, identity);
    } catch (InvalidArgumentException | IOException e) {
      throw new RuntimeException(e);
    }

    log.info("Successfully enrolled admin user '{}' and imported it into the wallet", ADMIN_NAME);
  }

  public Identity getIdentity() throws IOException {
    return wallet.get(ADMIN_NAME);
  }
}
