package com.softjourn.coins;

import com.softjourn.common.helper.*;
import java.util.HashMap;
import java.util.Map;
import org.hyperledger.fabric.contract.Context;

public class ContextHelperImpl extends ContextHelper {

  private final Map<String, Object> writeStateCache = new HashMap<>();
  private String id;
  private Long timestamp;

  @Override
  public <K, V> Map<? extends K, ? extends V> getMap(final Context ctx, final String key) {
    if (writeStateCache.containsKey(key)) {
      return (Map<K, V>) writeStateCache.get(key);
    }
    return new HashMap<>();
  }

  @Override
  public void writeMap(final Context ctx, final String key, final Map<?, ?> value) {
    writeStateCache.put(key, value);
  }

  @Override
  public <T> T getState(final Context ctx, final String key, Class<T> clazz) {
    return (T) writeStateCache.get(key);
  }

  @Override
  public String getNextId() {
    return id;
  }

  @Override
  public long getCurrentTimestamp() {
    return timestamp;
  }

  public void putState(String key, Object value) {
    writeStateCache.put(key, value);
  }

  public void setNextId(String id) {
    this.id = id;
  }

  public void setCurrentTimestamp(Long timestamp) {
    this.timestamp = timestamp;
  }
}
