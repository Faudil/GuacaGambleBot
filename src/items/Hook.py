from src.database.other import reset_user_limit
from src.items.Item import Item, ItemType, ItemRarity


class Hook(Item):
    def __init__(self):
        super().__init__(
            name="Hameçon",
            price=200,
            description="Attire les poissons ! Réinitialise le cooldown de !fish.",
            item_type=ItemType.consumable,
            rarity=ItemRarity.common
        )

    async def use(self, ctx, **kwargs):
        reset_user_limit(ctx.author.id, "fish")
        await ctx.send("☕ **Splash...** Les poissons arrivent ! Ton `!fish` est de nouveau disponible.")
        return True
