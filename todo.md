# Friss

# Go cuccok
- [viper](https://github.com/spf13/viper)
- [cobra](https://github.com/spf13/cobra)
- [yaml? -> viper/cobra lehet kezeli](https://github.com/goccy/go-yaml)
- [yaml? -> viper/cobra lehet kezeli](https://github.com/goccy/go-yaml)
- [prmethushoz kinyerni: styx](https://github.com/go-pluto/styx)

# Ötlet
.yaml file-ba lehetne definiálni a mérést (méréseket)

- Script készítése
  - Megadható indításnál, hogy offloadban, vagy ne offloadban menjen
  - Elindít (turncat) iperf küldi a loadot (legyen paramétezhető)
  - Lementjük, hogy mikor kezdődött a mérés
  - X idő után / inputra leáll, lement mikor végzett
  - styx?-el lekérni az adatokat prometheusból, lementeni őket
  - Lehetőleg több clusterre is lehessen végezni egyszerre (cluster ip-k felsorolása stb, utána külön fájlokba mentés)

# Következő todok

- Megvizsgálni BPF tool-al: [bpf-tool](https://bpftool.dev/)
- Több konfigurációval megnézni a méréseket
- Rendszerkihasználtságot nézni, hogy egyes megoldásoknál mekkora terhelések vannak.
- Félév végére legyen 1-2 mérés azért
- Prometheus API-ból kinyerés

## Scirp feladata

- Megnézi mennyi az idő
- Load generál
- Leáall
- Prometheusból kinyeri a cuccokat
- Összerak egy jó adathalmazt :)
- (Iperf helyett lesz egy load generator majd)
