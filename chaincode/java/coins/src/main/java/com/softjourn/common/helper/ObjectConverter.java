package com.softjourn.common.helper;

import com.owlike.genson.Genson;

public class ObjectConverter {

  private final Genson genson;

  public ObjectConverter() {
    genson = new Genson();
  }

  public byte[] serialize(Object o) {
    return genson.serializeBytes(o);
  }

  public <T> T deserialize(byte[] b, Class<T> c) {
    return genson.deserialize(b, c);
  }

  public <T> T deserialize(String json, Class<T> c) {
    return genson.deserialize(json, c);
  }
}
