package com.softjourn.coins.gateway.config;

import java.util.List;
import java.util.Map;
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
  private Map<String, CertificateAuthority> certificateAuthorities;
  private List<String> supportedChaincodes;
  private String defaultChaincode;

  @Getter
  @Setter
  public static class CertificateAuthority {

    private String url;
    private List<String> tlsCACerts;
  }
}
