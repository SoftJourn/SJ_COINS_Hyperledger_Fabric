package com.softjourn.coins;

import com.softjourn.common.helper.ContextHelper;
import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import org.hyperledger.fabric.contract.Context;

public class ContextHelperImpl extends ContextHelper {

  private final Map<String, Object> writeStateCache = new HashMap<>();
  private final List<String> nextIds = new ArrayList<>();

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
    Iterator<String> iterator = nextIds.iterator();
    if (!iterator.hasNext()) {
      throw new IllegalStateException("Next id list is empty");
    }

    String id = iterator.next();
    iterator.remove();
    return id;
  }

  @Override
  public long getCurrentTimestamp() {
    return timestamp;
  }

  public ContextHelperImpl putState(String key, Object value) {
    writeStateCache.put(key, value);
    return this;
  }

  public ContextHelperImpl setNextId(String id) {
    nextIds.add(id);
    return this;
  }

  public ContextHelperImpl setNextId(Collection<String> ids) {
    nextIds.addAll(ids);
    return this;
  }

  public ContextHelperImpl setCurrentTimestamp(Long timestamp) {
    this.timestamp = timestamp;
    return this;
  }
}
