// Instead of finding just one served version, iterate over all served versions
servedVersions := []string{}
for _, v := range crd.Spec.Versions {
    if v.Served {
        servedVersions = append(servedVersions, v.Name)
    }
}

// Skip this CRD if no served versions are found
if len(servedVersions) == 0 {
    continue
}

foundInstances := false

for _, version := range servedVersions {
    gvr := schema.GroupVersionResource{
        Group:    crd.Spec.Group,
        Version:  version,
        Resource: crd.Spec.Names.Plural,
    }
    instances, err := dynamicClient.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
    if err != nil {
        // If we get an error querying the resource, skip this version
        continue
    }
    if len(instances.Items) > 0 {
        foundInstances = true
        break
    }
}

if !foundInstances {
    reason := "CRD has no instances"
    unusedCRDs = append(unusedCRDs, ResourceInfo{Name: crd.Name, Reason: reason})
}