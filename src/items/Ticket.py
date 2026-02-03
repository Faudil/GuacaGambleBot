from src.data_handling import use_item_db, reset_user_limit
from src.items.Item import Item


class Ticket(Item):
    def __init__(self):
        super().__init__(
            name="ticket",
            price=100,
            description="Te donne une nouvelle chance, reset tes limites de casino et coinflip"
        )

    async def use(self, ctx, **kwargs):
        if not use_item_db(ctx.author.id, self.name):
            await ctx.send("❌ Tu n'as pas cet objet.")
            return False

        reset_user_limit(ctx.author.id, "coinflip")
        reset_user_limit(ctx.author.id, "slots")
        await ctx.send(" Tes `!coinflips` et `!casino` sont de nouveaux disponible ! Bonne chance !")
        return True