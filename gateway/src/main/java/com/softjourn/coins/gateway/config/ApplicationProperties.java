package com.softjourn.coins.gateway.config;

import java.util.List;
import java.util.Map;
import java.util.function.Function;
import java.util.stream.Collectors;
import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Configuration;

@Getter
@Setter
@Configuration
@ConfigurationProperties("application")
public class ApplicationProperties {

  private String caName;
  private String keyValueStore;
  private String adminUsername;
  private String adminPassword;
  private String mspId;
  private Map<String, CertificateAuthority> certificateAuthoritiesRaw;
  private Map<String, CertificateAuthority> certificateAuthorities;
  private List<String> supportedChaincodes;
  private String defaultChaincode;

  public Map<String, CertificateAuthority> getCertificateAuthorities() {
    if (certificateAuthorities == null) {
      certificateAuthorities = certificateAuthoritiesRaw.values().stream()
          .collect(Collectors.toMap(CertificateAuthority::getId, Function.identity()));
    }

    return certificateAuthorities;
  }

  @Getter
  @Setter
  public static class CertificateAuthority {

    private String id;
    private String url;
    private List<String> tlsCACerts;
  }
}
