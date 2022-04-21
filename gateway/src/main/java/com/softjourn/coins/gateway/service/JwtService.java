package com.softjourn.coins.gateway.service;

import io.jsonwebtoken.Jwts;
import io.jsonwebtoken.SignatureAlgorithm;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import javax.crypto.SecretKey;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
public class JwtService {

  private final static String USERNAME_KEY = "username";

  private final SecretKey secretKey;

  @Value("${spring.security.oauth2.resourceserver.jwt.expiration}")
  private int jwtExpiration;

  public String generateToken(String username, String orgName) {
    return Jwts.builder()
        .setClaims(new HashMap<>() {{
          put(USERNAME_KEY, username);
          put("orgName", orgName);
        }})
        .setIssuedAt(Date.from(Instant.now()))
        .setExpiration(Date.from(Instant.now().plus(jwtExpiration, ChronoUnit.SECONDS)))
        .signWith(SignatureAlgorithm.forName(secretKey.getAlgorithm()), secretKey)
        .compact();
  }

  public String getUsername(JwtAuthenticationToken token) {
    Map<String, Object> attributes = token.getTokenAttributes();
    return Optional.ofNullable(attributes.get(USERNAME_KEY))
        .map(String::valueOf)
        .orElseThrow(
            () -> new IllegalStateException("Authentication token desn't conain username"));
  }
}
