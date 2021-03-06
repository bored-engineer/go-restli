package io.papacharlie.gorestli;

import com.linkedin.restli.restspec.ActionSchema;
import com.linkedin.restli.restspec.ActionSchemaArray;
import com.linkedin.restli.restspec.CollectionSchema;
import com.linkedin.restli.restspec.FinderSchema;
import com.linkedin.restli.restspec.ResourceSchema;
import com.linkedin.restli.restspec.SimpleSchema;
import io.papacharlie.gorestli.json.Method.PathKey;
import io.papacharlie.gorestli.json.Resource;
import io.papacharlie.gorestli.json.RestliType;
import java.io.File;
import java.util.Collections;
import java.util.HashSet;
import java.util.List;
import java.util.Set;


public class ResourceParser {
  private final ResourceSchema _schema;
  private final String _rootResourceName;
  private final TypeParser _typeParser;
  private final List<String> _namespaceChain;
  private final List<PathKey> _pathKeys;
  private final String _resourceFilename;
  private final MethodParser _methodParser;

  private ResourceParser(ResourceSchema schema, String resourceFilename, String rootResourceName,
      TypeParser typeParser, List<String> namespaceChain, List<PathKey> pathKeys) {
    _schema = schema;
    _resourceFilename = resourceFilename;
    _rootResourceName = rootResourceName;
    _typeParser = typeParser;
    _namespaceChain = Utils.append(namespaceChain, schema.getName());
    _pathKeys = pathKeys;
    _methodParser = new MethodParser(_typeParser, _schema, _pathKeys);
  }

  private ResourceParser(ResourceParser parent, ResourceSchema subResource, PathKey pathKey) {
    this(subResource, parent._resourceFilename, parent._rootResourceName, parent._typeParser, parent._namespaceChain,
        pathKey == null ? parent._pathKeys : Utils.append(parent._pathKeys, pathKey));
  }

  public ResourceParser(ResourceSchema schema, File resourceFilename, TypeParser typeParser) {
    this(schema, resourceFilename.toString(), schema.getName(), typeParser,
        Collections.singletonList(schema.getNamespace()), Collections.emptyList());
  }

  public Set<Resource> parse() {
    if (_schema.hasCollection() && _schema.getCollection().getIdentifier().hasParams()) {
      Utils.log("Complex Key resources are not supported. Skipping %s and its children.%n", _namespaceChain);
      return Collections.emptySet();
    }

    Resource resource = newResource();
    MethodParser methodParser = new MethodParser(_typeParser, _schema, _pathKeys);

    Set<Resource> resourcesAndSubResources = new HashSet<>();
    resourcesAndSubResources.add(resource);

    if (_schema.getActionsSet() != null) {
      addActions(resource, _schema.getActionsSet().getActions(), false);
    }

    if (_schema.getSimple() != null) {
      SimpleSchema simple = _schema.getSimple();
      addActions(resource, simple.getActions(), false);
      addRestMethods(resource, simple.getSupports());

      for (ResourceSchema subResource : Utils.emptyIfNull(simple.getEntity().getSubresources())) {
        resourcesAndSubResources.addAll(new ResourceParser(this, subResource, null).parse());
      }
    }

    if (_schema.getCollection() != null) {
      CollectionSchema collection = _schema.getCollection();
      addActions(resource, collection.getActions(), false);
      addActions(resource, collection.getEntity().getActions(), true);
      addRestMethods(resource, collection.getSupports());

      for (FinderSchema finder : Utils.emptyIfNull(collection.getFinders())) {
        resource.addMethod(methodParser.newFinderMethod(finder));
      }

      PathKey pathKey = PathKey.forCollection(collection, _typeParser);
      for (ResourceSchema subResource : Utils.emptyIfNull(collection.getEntity().getSubresources())) {
        resourcesAndSubResources.addAll(new ResourceParser(this, subResource, pathKey).parse());
      }
    }

    return resourcesAndSubResources;
  }

  private Resource newResource() {
    RestliType resourceType = _schema.hasSchema()
        ? _typeParser.parseFromRestSpec(_schema.getSchema())
        : null;
    return new Resource(
        String.join(".", _namespaceChain),
        _schema.getDoc(),
        _resourceFilename,
        _rootResourceName,
        resourceType);
  }

  private void addRestMethods(Resource resource, List<String> restMethods) {
    for (String restMethod : Utils.emptyIfNull(restMethods)) {
      resource.addMethod(_methodParser.newRestMethod(restMethod));
    }
  }

  private void addActions(Resource resource, ActionSchemaArray actions, boolean onEntity) {
    for (ActionSchema action : Utils.emptyIfNull(actions)) {
      resource.addMethod(_methodParser.newActionMethod(action, onEntity));
    }
  }
}
