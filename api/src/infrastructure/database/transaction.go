package database

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sandbox-auth/src/ent"
)

type transactionClientKey struct{}

type requestTransaction struct {
	root *ent.Client
	tx   *ent.Tx
}

func ClientFromContext(ctx context.Context, fallback *ent.Client) (*ent.Client, error) {
	// リクエスト内で初めて DB を使うタイミングまでトランザクションの開始を遅らせる。
	transaction, ok := ctx.Value(transactionClientKey{}).(*requestTransaction)
	if !ok {
		return fallback, nil
	}
	if transaction.tx == nil {
		tx, err := transaction.root.Tx(ctx)
		if err != nil {
			return nil, err
		}
		transaction.tx = tx
	}
	return transaction.tx.Client(), nil
}

func (t *requestTransaction) Commit() error {
	if t.tx == nil {
		return nil
	}
	return t.tx.Commit()
}

func (t *requestTransaction) Rollback() error {
	if t.tx == nil {
		return nil
	}
	return t.tx.Rollback()
}

func TransactionMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ハンドラが正常終了したリクエストだけコミットし、失敗時はロールバックする。
		transaction := &requestTransaction{root: client}
		committed := false
		defer func() {
			if recovered := recover(); recovered != nil {
				_ = transaction.Rollback()
				panic(recovered)
			}
			if !committed {
				_ = transaction.Rollback()
			}
		}()

		ctx := context.WithValue(c.Request.Context(), transactionClientKey{}, transaction)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		if len(c.Errors) > 0 || c.Writer.Status() >= http.StatusBadRequest {
			return
		}

		if err := transaction.Commit(); err != nil {
			_ = c.Error(err)
			return
		}
		committed = true
	}
}
