package config

// BcryptCostProd es el factor de coste para bcrypt en producción y stage.
// cost 12 tarda ~300 ms en hardware moderno — aceptable para operaciones de escritura de contraseñas.
const BcryptCostProd = 12

// BcryptCostTests es el factor de coste para bcrypt en tests unitarios.
// cost 4 mantiene la funcionalidad sin impacto en velocidad del CI.
const BcryptCostTests = 4
