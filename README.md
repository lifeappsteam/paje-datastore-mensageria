# Passos
- Primeiro parâmetro
- - idProduto (fornecido pela Lifeapps) 
- - Opcionalmente adicionar `.filadesejada`, caso não informado será utilizada a fila padrão (`.pedidos`)
- Modo de execução, que pode ser um dos dois a seguir:
- - Ativo;
- - Passivo;
- O terceiro parâmetro depende do segundo parâmetro:
- - Caso ativo, deve ser o endereço do client onde o pedido será enviado;
- - Caso passivo, deve conter o número da porta onde o client realizará o pooling de pedidos.

# Exemplos da execução em linha de comando
  - Caso ativo: './paje-datastore-mensageria.1.0.0 [idProduto] ativo http://[seu ip]/[end-point]'
  - Caso passivo: './paje-datastore-mensageria.1.0.0 [idProduto] passivo [porta desejada]'Caso fila: './- paje-datastore-mensageria.1.0.0 [idProduto.filaDesejada] ativo http://[seu ip]/[end-point]'