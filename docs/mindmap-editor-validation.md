# Guia de Validação do Editor de Mapas Mentais (Fase 3B)

Este guia descreve os testes e validações para homologar o Editor Visual de Mapas Mentais integrado com o React Flow.

---

## 1. Validando Layout Automático e Compatibilidade (Fase 3A)

1. Crie um mapa conceitual na Fase 3A (pode ser com status `COMPLETED` mas sem dados de posicionamento `position` no JSON).
2. Acesse `/app/maps/:id` e clique no botão **👁️ Abrir Editor Visual**.
3. O frontend redirecionará para `/app/maps/:id/editor`.
4. O editor deve aplicar automaticamente o algoritmo de distribuição espacial em árvore horizontal (`calculateAutoLayout`) de modo que nenhum nó se sobreponha.
5. As conexões (`edges`) sem ID devem ser normalizadas com a máscara `edge-${source}-${target}` em memória.
6. A viewport deve ser inicializada na coordenada padrão `(x: 0, y: 0, zoom: 1)`.
7. O mapa deve exibir o badge verde **✓ Mapa salvo** porque as modificações ocorreram apenas na memória.
8. Ao mover qualquer nó, o badge deve alterar para o estado piscante **⚠️ Alterações não salvas**.

---

## 2. Testes de Modificação no Canvas

* **Arrastar Nós**:
  * Arraste qualquer caixa de nó no canvas do React Flow. O estado de "Alterações não salvas" deve ser ativado.
* **Editar Informações**:
  * Clique em um nó. O painel lateral **Propriedades do Nó** deve abrir.
  * Altere o título ou o conteúdo explicativo. Os dados da caixa devem ser atualizados em tempo real no canvas.
* **Criar Filho**:
  * Passe o mouse sobre qualquer nó e clique no botão **+** (Adicionar nó filho).
  * Um novo nó com ID dinâmico deve surgir posicionado ligeiramente à direita do pai, conectado por uma linha roxa.
* **Remover Nó (Subárvore)**:
  * Passe o mouse sobre um nó que contenha ramificações e clique no ícone de lixeira **🗑️**.
  * O sistema deve exibir o modal de confirmação: `"Este nó possui X subitens. Ao remover, todos eles também serão apagados. Deseja continuar?"`
  * Confirme a exclusão. O nó e todos os seus descendentes (assim como suas conexões) devem desaparecer do canvas.
  * Tente remover o nó raiz (`root`). O editor deve bloquear a ação e alertar: `"Não é possível remover o nó principal."`

---

## 3. Validação de Ciclos (Anti-Looping)

Para garantir integridade hierárquica e estrutural, criamos um algoritmo de busca de loops na subárvore (tanto no frontend quanto no backend).
Se o usuário (ou um payload forjado) tentar estruturar um ciclo (por exemplo, nó C apontando para B e B apontando para C):
1. O backend recusará a gravação retornando `400 Bad Request`.
2. O corpo da resposta especificará: `"Erro estrutural: o mapa mental contém ciclos direcionados."`

---

## 4. Persistência de Posições e Viewport
1. Com as alterações prontas (nós reposicionados ou ramificações adicionadas), clique no botão **Salvar Mapa** no menu superior.
2. A API receberá a requisição `PATCH /mindmaps/:id` salvando o `jsonData` atualizado no PostgreSQL.
3. O status de alteração deve mudar de volta para **✓ Mapa salvo**.
4. Recarregue a página (F5) ou saia da rota e retorne. Todas as posições de nós e o nível de zoom/zoom viewport (`viewport`) devem ser renderizados exatamente como salvos.

---

## 5. Validação Multiempresa (Segurança)

1. Faça login com o Usuário A (vinculado à Empresa A).
2. Tente forjar uma chamada `PATCH /mindmaps/<ID_DO_MAPA_DA_EMPRESA_B>` enviando um payload de modificação.
3. O backend deve recusar com `403 Forbidden` e o log de auditoria deve registrar a tentativa não autorizada.
