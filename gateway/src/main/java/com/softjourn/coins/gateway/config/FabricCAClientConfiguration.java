package com.softjourn.coins.gateway.config;

import com.softjourn.coins.gateway.config.ApplicationProperties.CertificateAuthority;
import java.net.MalformedURLException;
import java.util.Optional;
import java.util.Properties;
import lombok.RequiredArgsConstructor;
import org.hyperledger.fabric_ca.sdk.HFCAClient;
import org.hyperledger.fabric_ca.sdk.exception.InvalidArgumentException;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
@RequiredArgsConstructor
public class FabricCAClientConfiguration {

  private final ApplicationProperties applicationProperties;

  @Bean
  public HFCAClient hfcaClient() throws MalformedURLException, InvalidArgumentException {
    CertificateAuthority certificateAuthority = Optional.ofNullable(
        applicationProperties.getCertificateAuthorities().get(applicationProperties.getCaName()))
        .orElseThrow(() -> new IllegalStateException("CA configuration not found"));
    Properties properties = new Properties();
    properties.put("trustedRoots", certificateAuthority.getTlsCACerts().iterator().next());
    properties.put("verify", false);

    return HFCAClient.createNewInstance(
        applicationProperties.getCaName(), certificateAuthority.getUrl(), properties);
  }
}
