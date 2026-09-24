# spatial-firebase (`firestore-h3`)

`firestore-h3` es una librería en Go que extiende el SDK oficial de **Google Cloud Firestore** para añadir capacidades de indexación y búsqueda espacial de alto rendimiento utilizando el índice hexagonal jerárquico **H3 (Uber)**.

Permite realizar búsquedas por **Radio (Proximidad / KNN)** y **Polígonos (Geofencing / Point-in-Polygon)** de manera nativa sin sobrecargar el motor de Firestore ni realizar escaneos completos de colecciones.

---

## 🚀 Características

- **Indexación Multinivel**: Asigna resoluciones H3 (ej. `r6`, `r8`, `r10`) para realizar búsquedas tanto a nivel macro (zonas/ciudades) como micro (metros/calles).
- **Proximidad con Filtro Haversine**: Búsqueda por radio que elimina falsos positivos mediante un filtrado secundario de precisión en memoria.
- **Batching Automático**: Maneja automáticamente el límite estricto de 30 elementos en consultas `IN` de Firestore.
- **Deduplicación Transparente**: Agrupa resultados por ID de documento al consultar múltiples celdas H3.
- **Soporte para Firestore Emulator**: Diseñado para probarse fácilmente de forma local sin costos en GCP.

---

## 📦 Instalación

```bash
go get github.com/AlexG695/firestore-h3