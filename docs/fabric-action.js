// ZITADEL Action to add custom claim for Hyperledger Fabric CA enrollment
function setFabricPolicy(ctx, api) {
  enrollmentRequest = {
    id: 'replacedWithSubject',
    type: 'client',
    affiliation: 'org1.department1',
    attrs: [
      {
        name: "hf.Registrar.Roles",
        value: "client",
        ecert: true,
      }
    ]
  }
  api.v1.claims.setClaim('fabric', enrollmentRequest)
}