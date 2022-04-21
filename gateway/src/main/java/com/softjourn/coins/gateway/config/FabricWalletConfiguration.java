package com.softjourn.coins.gateway.config;

import java.io.IOException;
import java.nio.file.Path;
import java.nio.file.Paths;
import lombok.RequiredArgsConstructor;
import org.hyperledger.fabric.gateway.Wallet;
import org.hyperledger.fabric.gateway.Wallets;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
@RequiredArgsConstructor
public class FabricWalletConfiguration {

  private final ApplicationProperties applicationProperties;

  @Bean
  public Wallet wallet() throws IOException {
    Path walletDirectory = Paths.get(applicationProperties.getKeyValueStore());
    return Wallets.newFileSystemWallet(walletDirectory);
  }
}
